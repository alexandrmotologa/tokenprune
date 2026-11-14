package pruner

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

func TestTier1Syntactic(t *testing.T) {
	cfg := &config.Tier1SyntacticConfig{
		MinifyJSON:          true,
		StripHTMLComments:   true,
		NormalizeWhitespace: true,
		PruneTablePadding:   true,
	}
	counter := tokenizer.NewCounter(100)
	t1 := NewTier1Syntactic(cfg, counter)

	raw := `Here is the response:
<!-- debug internal log -->
{
  "status": "ok",
  "items": [
    1,
    2,
    3
  ]
}

|  Column A  |   Column B   |
|------------|--------------|
|    val1    |     val2     |

` + "```python\ndef hello():\n    # Preserve indentation\n    return 42\n```"

	msg := Message{
		Role:    "assistant",
		Content: raw,
	}

	diffs := t1.ProcessMessage(&msg, "gpt-4o")
	if len(diffs) == 0 {
		t.Fatal("expected diffs from tier 1 minification")
	}

	// Verify HTML comment was stripped
	if strings.Contains(msg.Content, "debug internal log") {
		t.Fatal("expected HTML comment to be stripped")
	}

	// Verify code indentation was preserved
	if !strings.Contains(msg.Content, "    return 42") {
		t.Fatal("expected python code block indentation to be strictly preserved")
	}

	// Verify table padding was pruned
	if strings.Contains(msg.Content, "|   Column B   |") {
		t.Fatal("expected table padding to be pruned")
	}
}

func TestTier2ToolEviction(t *testing.T) {
	cfg := &config.Tier2ToolEvictionConfig{
		Enabled:               true,
		EvictSupersededReads:  true,
		MaxToolOutputAgeTurns: 2,
		ReceiptTemplate:       "[Receipt: Tool '%s' (id: %s) output condensed (%d lines, %d bytes). Superseded by subsequent turn.]",
		PreserveErrors:        true,
		ProtectedToolNames:    []string{"deploy"},
	}
	counter := tokenizer.NewCounter(100)
	t2 := NewTier2ToolEviction(cfg, counter)

	longDump := strings.Repeat("Line of raw file content from old read_file call.\n", 30) // ~1500 chars

	messages := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_read_1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "read_file",
						Arguments: `{"path": "pkg/server.go"}`,
					},
				},
			},
		},
		{
			Role:       "tool",
			Name:       "read_file",
			ToolCallID: "call_read_1",
			Content:    longDump,
		},
		{
			Role:    "assistant",
			Content: "Analyzed server.go. Now editing.",
		},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_write_1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "write_file",
						Arguments: `{"path": "pkg/server.go"}`,
					},
				},
			},
		},
		{
			Role:       "tool",
			Name:       "write_file",
			ToolCallID: "call_write_1",
			Content:    "Successfully wrote 24 lines to pkg/server.go",
		},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_err_1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "bash",
						Arguments: `{"command": "go test ./..."}`,
					},
				},
			},
		},
		{
			Role:       "tool",
			Name:       "bash",
			ToolCallID: "call_err_1",
			Content:    "Error: tests failed with exit status 1 in server_test.go:42\n" + strings.Repeat("trace line\n", 15),
		},
	}

	diffs, count := t2.ProcessMessages(messages, "claude-3-5-sonnet")
	if count < 1 {
		t.Fatalf("expected at least 1 tool eviction, got %d", count)
	}
	if len(diffs) < 1 {
		t.Fatal("expected diff items for evicted tool")
	}

	// Verify call_read_1 was replaced with receipt
	if !strings.Contains(messages[1].Content, "[Receipt: Tool 'read_file'") {
		t.Fatalf("expected read_file to have receipt, got: %s", messages[1].Content)
	}

	// Verify error tool (call_err_1) was PRESERVED
	if !strings.Contains(messages[6].Content, "Error: tests failed") {
		t.Fatal("expected error tool output to be preserved")
	}
}

func TestTier3StateDelta(t *testing.T) {
	cfg := &config.Tier3SemanticDeltaConfig{
		Enabled:            true,
		CompressAfterTurns: 4,
		KeepRecentTurns:    2,
		ProtectedKeywords:  []string{"CRITICAL_RULES"},
	}
	counter := tokenizer.NewCounter(100)
	t3 := NewTier3StateDelta(cfg, counter)

	messages := []Message{
		{Role: "system", Content: "You are an AI software architect. CRITICAL_RULES: Never break tests."},
		{Role: "user", Content: "Build the tokenprune system."},
		{Role: "assistant", Content: "Understood. Starting module setup."},
		{Role: "user", Content: "Proceed with step 1."},
		{Role: "assistant", Content: "Step 1 complete. Now starting step 2."},
		{Role: "user", Content: "Proceed with step 2."},
		{Role: "assistant", Content: "Step 2 complete."},
		{Role: "user", Content: "What is next?"},
	}

	out, diffs, count := t3.ProcessMessages(messages, "gpt-4o")
	if count == 0 {
		t.Fatal("expected turns to be consolidated in tier 3")
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs from tier 3")
	}

	// System message must remain at index 0
	if out[0].Role != "system" || !strings.Contains(out[0].Content, "CRITICAL_RULES") {
		t.Fatal("expected system message to remain intact")
	}

	// Recent message must be preserved
	lastMsg := out[len(out)-1]
	if lastMsg.Content != "What is next?" {
		t.Fatalf("expected last message to be 'What is next?', got: %s", lastMsg.Content)
	}
}

func TestFullPipeline(t *testing.T) {
	cfg := config.DefaultConfig()
	pipeline := NewPipeline(&cfg.Pruning, tokenizer.NewCounter(1000))

	longDump := strings.Repeat("Raw tool output dump with lots of redundant tokens.\n", 50)

	messages := []Message{
		{Role: "system", Content: "You are an assistant."},
		{Role: "user", Content: "Please inspect files."},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "read_file",
						Arguments: "{\n  \"path\": \"config.go\"\n}",
					},
				},
			},
		},
		{
			Role:       "tool",
			Name:       "read_file",
			ToolCallID: "call_1",
			Content:    longDump,
		},
		{Role: "assistant", Content: "File inspected. Ready for next step."},
		{Role: "user", Content: "Continue."},
		{Role: "assistant", Content: "Done."},
	}

	pruned, stats := pipeline.Prune(messages, "claude-3-5-sonnet")

	if stats.InboundTokens <= stats.OutboundTokens {
		t.Fatalf("expected savings, inbound: %d, outbound: %d", stats.InboundTokens, stats.OutboundTokens)
	}

	if stats.TokensSaved <= 0 {
		t.Fatalf("expected tokens saved > 0, got %d", stats.TokensSaved)
	}

	if stats.CompressionRatio <= 0 {
		t.Fatalf("expected compression ratio > 0, got %f", stats.CompressionRatio)
	}

	if stats.ProcessingLatencyMs > 50.0 {
		t.Fatalf("processing latency too high: %f ms", stats.ProcessingLatencyMs)
	}

	if len(pruned) == 0 {
		t.Fatal("pruned messages should not be empty")
	}
}


func BenchmarkPrunerPipeline(b *testing.B) {
	cfg := config.DefaultConfig()
	pipeline := NewPipeline(&cfg.Pruning, tokenizer.NewCounter(5000))

	longDump := strings.Repeat("Raw tool output dump with lots of redundant tokens.\n", 30)

	messages := []Message{
		{Role: "system", Content: "You are an assistant."},
		{Role: "user", Content: "Please inspect files."},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "read_file",
						Arguments: "{\n  \"path\": \"config.go\"\n}",
					},
				},
			},
		},
		{
			Role:       "tool",
			Name:       "read_file",
			ToolCallID: "call_1",
			Content:    longDump,
		},
		{Role: "assistant", Content: "File inspected. Ready for next step."},
		{Role: "user", Content: "Continue."},
		{Role: "assistant", Content: "Done."},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Prune(messages, "claude-3-5-sonnet")
	}
}
