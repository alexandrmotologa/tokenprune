package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/store"
)

func main() {
	dbPath := "tokenprune_telemetry.db"
	_ = os.Remove(dbPath)

	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	sessions := []struct {
		id        string
		sessionID string
		provider  string
		model     string
		inbound   int
		outbound  int
		saved     int
		t1        int
		t2        int
		t3        int
		evicted   int
		turns     int
		latency   float64
		ttft      float64
		cost      float64
		raw       string
		pruned    string
		age       time.Duration
	}{
		{
			id:        "req_cl35_01",
			sessionID: "sess_claude_refactor_pool",
			provider:  "anthropic",
			model:     "claude-3-5-sonnet",
			inbound:   4200,
			outbound:  1420,
			saved:     2780,
			t1:        520,
			t2:        1840,
			t3:        420,
			evicted:   2,
			turns:     1,
			latency:   0.42,
			ttft:      210.0,
			cost:      0.00834,
			raw:       `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Inspect database connection pool config."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"read_file","input":{"path":"internal/db/pool.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"package db\n\n// 80 lines of pool config..."}]}]}`,
			pruned:    `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Inspect database connection pool config."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"read_file","input":{"path":"internal/db/pool.go"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"[tool_result: read_file internal/db/pool.go (80 lines) preserved in state delta]"}]}]}`,
			age:       35 * time.Minute,
		},
		{
			id:        "req_cl35_02",
			sessionID: "sess_claude_refactor_pool",
			provider:  "anthropic",
			model:     "claude-3-5-sonnet",
			inbound:   8140,
			outbound:  2310,
			saved:     5830,
			t1:        910,
			t2:        3920,
			t3:        1000,
			evicted:   3,
			turns:     2,
			latency:   0.65,
			ttft:      245.0,
			cost:      0.01749,
			raw:       `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Update max idle connections to 25."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_2","name":"edit_file","input":{"path":"internal/db/pool.go"}}]}]}`,
			pruned:    `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Update max idle connections to 25."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_2","name":"edit_file","input":{"path":"internal/db/pool.go"}}]}]}`,
			age:       30 * time.Minute,
		},
		{
			id:        "req_cl35_03",
			sessionID: "sess_claude_refactor_pool",
			provider:  "anthropic",
			model:     "claude-3-5-sonnet",
			inbound:   14800,
			outbound:  3920,
			saved:     10880,
			t1:        1450,
			t2:        7800,
			t3:        1630,
			evicted:   4,
			turns:     3,
			latency:   0.91,
			ttft:      280.0,
			cost:      0.03264,
			raw:       `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Run unit tests on pool."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_3","name":"run_command","input":{"CommandLine":"go test ./internal/db"}}]}]}`,
			pruned:    `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Run unit tests on pool."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_3","name":"run_command","input":{"CommandLine":"go test ./internal/db"}}]}]}`,
			age:       25 * time.Minute,
		},
		{
			id:        "req_gpt4o_01",
			sessionID: "sess_aider_test_suite",
			provider:  "openai",
			model:     "gpt-4o",
			inbound:   6100,
			outbound:  1980,
			saved:     4120,
			t1:        720,
			t2:        2800,
			t3:        600,
			evicted:   2,
			turns:     1,
			latency:   0.55,
			ttft:      220.0,
			cost:      0.01030,
			raw:       `{"model":"gpt-4o","messages":[{"role":"system","content":"You are a software architect. CRITICAL_RULES: Maintain test invariants."},{"role":"user","content":"Check syntax in tokenizer cache."}]}`,
			pruned:    `{"model":"gpt-4o","messages":[{"role":"system","content":"You are a software architect. CRITICAL_RULES: Maintain test invariants."},{"role":"user","content":"Check syntax in tokenizer cache."}]}`,
			age:       22 * time.Minute,
		},
		{
			id:        "req_gpt4o_02",
			sessionID: "sess_aider_test_suite",
			provider:  "openai",
			model:     "gpt-4o",
			inbound:   12300,
			outbound:  3450,
			saved:     8850,
			t1:        1320,
			t2:        6100,
			t3:        1430,
			evicted:   4,
			turns:     2,
			latency:   0.88,
			ttft:      290.0,
			cost:      0.02213,
			raw:       `{"model":"gpt-4o","messages":[{"role":"user","content":"Rebuild tokenizer benchmarks."}]}`,
			pruned:    `{"model":"gpt-4o","messages":[{"role":"user","content":"Rebuild tokenizer benchmarks."}]}`,
			age:       18 * time.Minute,
		},
		{
			id:        "req_gpt4o_03",
			sessionID: "sess_aider_test_suite",
			provider:  "openai",
			model:     "gpt-4o",
			inbound:   19800,
			outbound:  5120,
			saved:     14680,
			t1:        1850,
			t2:        10200,
			t3:        2630,
			evicted:   6,
			turns:     3,
			latency:   1.05,
			ttft:      340.0,
			cost:      0.03670,
			raw:       `{"model":"gpt-4o","messages":[{"role":"user","content":"Run benchmark comparison with tiktoken."}]}`,
			pruned:    `{"model":"gpt-4o","messages":[{"role":"user","content":"Run benchmark comparison with tiktoken."}]}`,
			age:       15 * time.Minute,
		},
		{
			id:        "req_cl37_01",
			sessionID: "sess_cursor_agent_migration",
			provider:  "anthropic",
			model:     "claude-3-7-sonnet",
			inbound:   5400,
			outbound:  1650,
			saved:     3750,
			t1:        680,
			t2:        2450,
			t3:        620,
			evicted:   2,
			turns:     1,
			latency:   0.48,
			ttft:      230.0,
			cost:      0.01125,
			raw:       `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Scan repo dependencies for deprecated modernc sqlite calls."}]}`,
			pruned:    `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Scan repo dependencies for deprecated modernc sqlite calls."}]}`,
			age:       12 * time.Minute,
		},
		{
			id:        "req_cl37_02",
			sessionID: "sess_cursor_agent_migration",
			provider:  "anthropic",
			model:     "claude-3-7-sonnet",
			inbound:   11200,
			outbound:  2980,
			saved:     8220,
			t1:        1250,
			t2:        5600,
			t3:        1370,
			evicted:   3,
			turns:     2,
			latency:   0.79,
			ttft:      275.0,
			cost:      0.02466,
			raw:       `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Inspect migration statements."}]}`,
			pruned:    `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Inspect migration statements."}]}`,
			age:       9 * time.Minute,
		},
		{
			id:        "req_cl37_03",
			sessionID: "sess_cursor_agent_migration",
			provider:  "anthropic",
			model:     "claude-3-7-sonnet",
			inbound:   18900,
			outbound:  4820,
			saved:     14080,
			t1:        1820,
			t2:        9950,
			t3:        2310,
			evicted:   5,
			turns:     3,
			latency:   1.02,
			ttft:      330.0,
			cost:      0.04224,
			raw:       `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Audit indexes on created_at."}]}`,
			pruned:    `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Audit indexes on created_at."}]}`,
			age:       7 * time.Minute,
		},
		{
			id:        "req_cl37_04",
			sessionID: "sess_cursor_agent_migration",
			provider:  "anthropic",
			model:     "claude-3-7-sonnet",
			inbound:   26400,
			outbound:  6700,
			saved:     19700,
			t1:        2410,
			t2:        14100,
			t3:        3190,
			evicted:   7,
			turns:     4,
			latency:   1.25,
			ttft:      380.0,
			cost:      0.05910,
			raw:       `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Optimize pragma journal_mode=WAL settings."}]}`,
			pruned:    `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Optimize pragma journal_mode=WAL settings."}]}`,
			age:       5 * time.Minute,
		},
		{
			id:        "req_cl35_04",
			sessionID: "sess_claude_refactor_pool",
			provider:  "anthropic",
			model:     "claude-3-5-sonnet",
			inbound:   22400,
			outbound:  5610,
			saved:     16790,
			t1:        2100,
			t2:        11900,
			t3:        2790,
			evicted:   6,
			turns:     4,
			latency:   1.15,
			ttft:      320.0,
			cost:      0.05037,
			raw:       `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Check latency histogram after pool adjustments."}]}`,
			pruned:    `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Check latency histogram after pool adjustments."}]}`,
			age:       3 * time.Minute,
		},
		{
			id:        "req_cl35_05",
			sessionID: "sess_claude_refactor_pool",
			provider:  "anthropic",
			model:     "claude-3-5-sonnet",
			inbound:   31500,
			outbound:  7890,
			saved:     23610,
			t1:        2950,
			t2:        16800,
			t3:        3860,
			evicted:   8,
			turns:     5,
			latency:   1.34,
			ttft:      360.0,
			cost:      0.07083,
			raw:       `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Verify zero-allocation tokenizer caching."}]}`,
			pruned:    `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Verify zero-allocation tokenizer caching."}]}`,
			age:       1 * time.Minute,
		},
		{
			id:        "req_cl35_06",
			sessionID: "sess_claude_refactor_pool",
			provider:  "anthropic",
			model:     "claude-3-5-sonnet",
			inbound:   42100,
			outbound:  10240,
			saved:     31860,
			t1:        3820,
			t2:        22900,
			t3:        5140,
			evicted:   11,
			turns:     6,
			latency:   1.62,
			ttft:      410.0,
			cost:      0.09558,
			raw:       `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Finalize end-to-end load test under 100 concurrent streams."}]}`,
			pruned:    `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"Finalize end-to-end load test under 100 concurrent streams."}]}`,
			age:       15 * time.Second,
		},
	}

	for _, s := range sessions {
		ratio := float64(s.saved) / float64(s.inbound) * 100.0
		rec := &store.SessionRecord{
			ID:                  s.id,
			SessionID:           s.sessionID,
			CreatedAt:           now.Add(-s.age),
			Provider:            s.provider,
			Model:               s.model,
			InboundTokens:       s.inbound,
			OutboundTokens:      s.outbound,
			TokensSaved:         s.saved,
			CompressionRatio:    ratio,
			Tier1Saved:          s.t1,
			Tier2Saved:          s.t2,
			Tier3Saved:          s.t3,
			EvictedToolsCount:   s.evicted,
			ConsolidatedTurns:   s.turns,
			ProcessingLatencyMs: s.latency,
			UpstreamTTFTMs:      s.ttft,
			USDSaved:            s.cost,
			RawPreview:          s.raw,
			PrunedPreview:       s.pruned,
		}
		if err := db.Record(rec); err != nil {
			log.Fatalf("failed to insert record %s: %v", s.id, err)
		}
	}

	stats, err := db.GetStats()
	if err != nil {
		log.Fatalf("failed to query stats: %v", err)
	}

	fmt.Printf("Seeded %d sessions into %s successfully!\n", len(sessions), dbPath)
	fmt.Printf("Total Requests: %d | Tokens Saved: %d | Ratio: %.1f%% | Cost Avoided: $%.3f\n",
		stats.TotalRequests, stats.TotalTokensSaved, stats.AvgCompressionRatio, stats.TotalUSDSaved)
}
