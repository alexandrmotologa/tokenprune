package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/store"
)

func runDump(args []string) {
	fs := flag.NewFlagSet("dump", flag.ExitOnError)
	dbPath := fs.String("db", "./tokenprune.db", "Path to SQLite database")
	format := fs.String("format", "jsonl", "Export format (jsonl, csv)")
	outFile := fs.String("out", "", "Output file path (defaults to stdout)")
	limit := fs.Int("limit", 1000, "Maximum number of records to export")
	_ = fs.Parse(args)

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	sessions, err := db.ListSessions(*limit, 0)
	if err != nil {
		log.Fatalf("failed to query sessions: %v", err)
	}

	var out io.Writer = os.Stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			log.Fatalf("failed to create output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	if *format == "csv" {
		w := csv.NewWriter(out)
		defer w.Flush()

		// Write header
		_ = w.Write([]string{
			"id", "session_id", "created_at", "provider", "model",
			"inbound_tokens", "outbound_tokens", "tokens_saved", "compression_ratio",
			"processing_latency_ms", "upstream_ttft_ms", "usd_saved",
		})

		for _, s := range sessions {
			_ = w.Write([]string{
				s.ID,
				s.SessionID,
				s.CreatedAt.Format(time.RFC3339),
				s.Provider,
				s.Model,
				strconv.Itoa(s.InboundTokens),
				strconv.Itoa(s.OutboundTokens),
				strconv.Itoa(s.TokensSaved),
				fmt.Sprintf("%.2f", s.CompressionRatio),
				fmt.Sprintf("%.2f", s.ProcessingLatencyMs),
				fmt.Sprintf("%.2f", s.UpstreamTTFTMs),
				fmt.Sprintf("%.6f", s.USDSaved),
			})
		}
	} else {
		// Default JSON Lines
		encoder := json.NewEncoder(out)
		for _, s := range sessions {
			_ = encoder.Encode(s)
		}
	}
}
