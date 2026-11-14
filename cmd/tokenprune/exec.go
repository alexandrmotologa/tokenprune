package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/api"
	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/proxy"
	"github.com/alexandrmotologa/tokenprune/internal/store"
)

func runExec(args []string) {
	// Separate tokenprune flags from the child command after "--"
	var tpArgs []string
	var childCmd []string

	splitIdx := -1
	for i, arg := range args {
		if arg == "--" {
			splitIdx = i
			break
		}
	}

	if splitIdx >= 0 {
		tpArgs = args[:splitIdx]
		childCmd = args[splitIdx+1:]
	} else {
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			childCmd = args
		} else {
			tpArgs = args
		}
	}

	if len(childCmd) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: tokenprune exec [flags] -- <command> [args...]\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n  tokenprune exec -- claude\n  tokenprune exec -- aider --model claude-3-5-sonnet\n")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("exec", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to tokenprune.yaml")
	provider := fs.String("provider", "anthropic", "Upstream provider (anthropic, openai, gemini)")
	_ = fs.Parse(tpArgs)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
	if *provider != "" {
		cfg.Server.TargetProvider = *provider
	}
	cfg.ValidateAndNormalize()

	// 1. Find open ephemeral port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to allocate local port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Server.Listen = ":" + strconv.Itoa(port)

	// 2. Open temporary in-memory store for this exec session
	db, err := store.Open(":memory:")
	if err != nil {
		log.Fatalf("failed to initialize session store: %v", err)
	}
	defer db.Close()

	proxyServer := proxy.NewServer(cfg, db)
	apiHandler := api.NewHandler(cfg, db)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)
	mux.Handle("/v1/", proxyServer.Handler())
	mux.Handle("/", proxyServer.Handler())

	server := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: mux,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	proxyBaseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	fmt.Printf("[tokenprune] Launching transparent proxy on %s (provider: %s)\n", proxyBaseURL, cfg.Server.TargetProvider)

	// 3. Prepare child process environment
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("ANTHROPIC_BASE_URL=%s", proxyBaseURL),
		fmt.Sprintf("OPENAI_BASE_URL=%s/v1", proxyBaseURL),
	)

	// 4. Spawn child process
	cmd := exec.Command(childCmd[0], childCmd[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward termination signals to child process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	cmdErr := cmd.Run()

	// 5. Shutdown proxy
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// 6. Print summary report
	stats, err := db.GetStats()
	if err == nil && stats.TotalRequests > 0 {
		fmt.Printf("\n┌────────────────────────────────────────────────────────┐\n")
		fmt.Printf("│  tokenprune Session Summary                            │\n")
		fmt.Printf("├────────────────────────────────────────────────────────┤\n")
		fmt.Printf("│  Requests Intercepted: %-32d│\n", stats.TotalRequests)
		fmt.Printf("│  Prompt Tokens Saved:  %-32d│\n", stats.TotalTokensSaved)
		fmt.Printf("│  Compression Ratio:    %-31.1f%%│\n", stats.AvgCompressionRatio)
		fmt.Printf("│  Direct Cost Avoided:  $%-31.4f│\n", stats.TotalUSDSaved)
		fmt.Printf("└────────────────────────────────────────────────────────┘\n\n")
	}

	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
