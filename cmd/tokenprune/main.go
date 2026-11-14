package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/api"
	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/pruner"
	"github.com/alexandrmotologa/tokenprune/internal/proxy"
	"github.com/alexandrmotologa/tokenprune/internal/store"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
	"github.com/alexandrmotologa/tokenprune/web"
)

var (
	Version   = "1.0.0"
	BuildTime = time.Now().Format("2006-01-02")
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		runServe(os.Args[2:])
	case "bench":
		runBench(os.Args[2:])
	case "prune":
		runPrune(os.Args[2:])
	case "report":
		runReport(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("tokenprune version %s (built %s, %s/%s)\n", Version, BuildTime, runtime.GOOS, runtime.GOARCH)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`tokenprune: Real-Time Semantic Context Compression & Streaming Reverse Proxy

Usage:
  tokenprune [command] [flags]

Commands:
  serve      Start the reverse proxy and embedded Telemetry Studio
  bench      Benchmark context compression and latency on a prompt payload
  prune      Pipe mode: compress raw context or JSON from stdin to stdout
  report     Display historical savings and performance telemetry
  version    Display version and build information

Run 'tokenprune [command] --help' for details on flags.`)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to tokenprune.yaml configuration file")
	port := fs.String("port", "", "Port to listen on (e.g. 8080 or :8080)")
	provider := fs.String("provider", "", "Target upstream provider (anthropic, openai, gemini)")
	upstream := fs.String("upstream", "", "Upstream provider base URL")
	dbPath := fs.String("db", "", "Path to SQLite database file")
	_ = fs.Parse(args)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	if *port != "" {
		if !strings.HasPrefix(*port, ":") {
			*port = ":" + *port
		}
		cfg.Server.Listen = *port
	}
	if *provider != "" {
		cfg.Server.TargetProvider = *provider
	}
	if *upstream != "" {
		cfg.Server.UpstreamURL = *upstream
	}
	if *dbPath != "" {
		cfg.Server.SQLitePath = *dbPath
	}
	cfg.ValidateAndNormalize()

	// Initialize database
	db, err := store.Open(cfg.Server.SQLitePath)
	if err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}
	defer db.Close()

	// Initialize proxy and API
	proxyServer := proxy.NewServer(cfg, db)
	apiHandler := api.NewHandler(cfg, db)
	webStudio := web.Handler()

	mux := http.NewServeMux()

	// Register API endpoints
	apiHandler.RegisterRoutes(mux)

	// Register Proxy handler for /v1/* endpoints
	mux.Handle("/v1/", proxyServer.Handler())

	// Fallback to embedded React Telemetry Studio
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If Anthropic /messages endpoint called on root path
		if strings.HasPrefix(r.URL.Path, "/messages") {
			proxyServer.ServeHTTP(w, r)
			return
		}
		webStudio.ServeHTTP(w, r)
	}))

	server := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	fmt.Printf("\n┌────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│  tokenprune v%s — High-Speed Semantic Context Proxy   │\n", Version)
	fmt.Printf("├────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│  Listening:        http://localhost%s               │\n", cfg.Server.Listen)
	fmt.Printf("│  Telemetry Studio: http://localhost%s/              │\n", cfg.Server.Listen)
	fmt.Printf("│  Target Provider:  %-36s│\n", cfg.Server.TargetProvider)
	fmt.Printf("│  Upstream URL:     %-36s│\n", cfg.Server.UpstreamURL)
	fmt.Printf("│  SQLite Database:  %-36s│\n", cfg.Server.SQLitePath)
	fmt.Printf("└────────────────────────────────────────────────────────┘\n\n")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server terminated: %v", err)
	}
}

func runBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	inputFile := fs.String("input", "", "Path to prompt payload JSON file")
	model := fs.String("model", "claude-3-5-sonnet", "Model name for token estimation and pricing")
	targetRatio := fs.Float64("ratio", 0.60, "Target compression ratio (0.1 to 0.8)")
	_ = fs.Parse(args)

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "error: --input file is required\n")
		os.Exit(1)
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("failed to read input file: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Pruning.TargetCompressionRatio = *targetRatio
	counter := tokenizer.NewCounter(5000)
	pipeline := pruner.NewPipeline(&cfg.Pruning, counter)

	start := time.Now()
	var pruned []byte
	var stats pruner.PruningStats

	if strings.Contains(string(data), `"messages"`) {
		pruned, _, stats, err = proxy.PruneAnthropicPayload(data, pipeline)
		if err != nil {
			pruned, _, stats, err = proxy.PruneOpenAIPayload(data, pipeline)
		}
	} else {
		msgs := []pruner.Message{{Role: "user", Content: string(data)}}
		_, stats = pipeline.Prune(msgs, *model)
		pruned = data
	}

	elapsed := time.Since(start)

	fmt.Printf("\n=== tokenprune Benchmark Results ===\n")
	fmt.Printf("Target Model:         %s\n", *model)
	fmt.Printf("Inbound Tokens:       %d tokens\n", stats.InboundTokens)
	fmt.Printf("Outbound Tokens:      %d tokens\n", stats.OutboundTokens)
	fmt.Printf("Tokens Saved:         %d tokens\n", stats.TokensSaved)
	fmt.Printf("Compression Ratio:    %.2f%%\n", stats.CompressionRatio)
	fmt.Printf("Tier 1 (Syntactic):   %d tokens saved\n", stats.Tier1TokensSaved)
	fmt.Printf("Tier 2 (Evictions):   %d tokens saved (%d tools)\n", stats.Tier2TokensSaved, stats.EvictedToolsCount)
	fmt.Printf("Tier 3 (State Delta): %d tokens saved (%d turns)\n", stats.Tier3TokensSaved, stats.ConsolidatedTurns)
	fmt.Printf("Total Benchmark Time: %v (Pipeline Latency: %.2f ms)\n", elapsed, stats.ProcessingLatencyMs)
	fmt.Printf("Payload Size:         %d bytes -> %d bytes\n\n", len(data), len(pruned))
}

func runPrune(args []string) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	model := fs.String("model", "claude-3-5-sonnet", "Model name")
	ratio := fs.Float64("ratio", 0.60, "Target compression ratio")
	_ = fs.Parse(args)

	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Pruning.TargetCompressionRatio = *ratio
	counter := tokenizer.NewCounter(5000)
	pipeline := pruner.NewPipeline(&cfg.Pruning, counter)

	trimmed := strings.TrimSpace(string(inputData))
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"messages"`) {
		pruned, _, _, err := proxy.PruneAnthropicPayload(inputData, pipeline)
		if err == nil {
			os.Stdout.Write(pruned)
			return
		}
		pruned, _, _, err = proxy.PruneOpenAIPayload(inputData, pipeline)
		if err == nil {
			os.Stdout.Write(pruned)
			return
		}
	}

	// Raw text fallback
	msgs := []pruner.Message{{Role: "user", Content: string(inputData)}}
	prunedMsgs, _ := pipeline.Prune(msgs, *model)
	if len(prunedMsgs) > 0 {
		fmt.Print(prunedMsgs[0].Content)
	} else {
		os.Stdout.Write(inputData)
	}
}

func runReport(args []string) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	dbPath := fs.String("db", "./tokenprune.db", "Path to SQLite database")
	format := fs.String("format", "table", "Output format (table, json)")
	_ = fs.Parse(args)

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		log.Fatalf("failed to query stats: %v", err)
	}

	if *format == "json" {
		_ = json.NewEncoder(os.Stdout).Encode(stats)
		return
	}

	fmt.Printf("\n=== tokenprune Cumulative Savings Report ===\n")
	fmt.Printf("Total Requests Processed: %d\n", stats.TotalRequests)
	fmt.Printf("Total Inbound Tokens:     %d\n", stats.TotalInboundTokens)
	fmt.Printf("Total Outbound Tokens:    %d\n", stats.TotalOutboundTokens)
	fmt.Printf("Total Tokens Saved:       %d\n", stats.TotalTokensSaved)
	fmt.Printf("Average Compression:      %.2f%%\n", stats.AvgCompressionRatio)
	fmt.Printf("Total Cost Avoidance:     $%.4f USD\n", stats.TotalUSDSaved)
	fmt.Printf("Average Pruning Latency:  %.2f ms\n", stats.AvgLatencyMs)
	fmt.Printf("Average Upstream TTFT:    %.1f ms\n\n", stats.AvgTTFTMs)
}
