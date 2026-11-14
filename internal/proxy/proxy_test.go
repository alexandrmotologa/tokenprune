package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/pruner"
	"github.com/alexandrmotologa/tokenprune/internal/store"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

func TestOpenAIAdapterPruning(t *testing.T) {
	cfg := config.DefaultConfig()
	counter := tokenizer.NewCounter(100)
	pipeline := pruner.NewPipeline(&cfg.Pruning, counter)

	rawPayload := []byte(`{
		"model": "gpt-4o",
		"messages": [
			{"role": "system", "content": "You are a coding assistant."},
			{"role": "user", "content": "Read the file."},
			{
				"role": "assistant",
				"tool_calls": [
					{
						"id": "call_123",
						"type": "function",
						"function": {
							"name": "read_file",
							"arguments": "{\n  \"path\": \"server.go\"\n}"
						}
					}
				]
			},
			{
				"role": "tool",
				"tool_call_id": "call_123",
				"name": "read_file",
				"content": "Line 1 of server.go\nLine 2 of server.go\nLine 3 of server.go\n"
			}
		]
	}`)

	pruned, model, stats, err := PruneOpenAIPayload(rawPayload, pipeline)
	if err != nil {
		t.Fatalf("failed to prune openai payload: %v", err)
	}

	if model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", model)
	}

	if len(pruned) == 0 {
		t.Fatal("pruned body is empty")
	}

	var res map[string]any
	if err := json.Unmarshal(pruned, &res); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if stats.InboundTokens <= 0 {
		t.Fatalf("expected positive inbound tokens, got %d", stats.InboundTokens)
	}
}

func TestAnthropicAdapterPruning(t *testing.T) {
	cfg := config.DefaultConfig()
	counter := tokenizer.NewCounter(100)
	pipeline := pruner.NewPipeline(&cfg.Pruning, counter)

	rawPayload := []byte(`{
		"model": "claude-3-5-sonnet",
		"system": "You are a software engineer.",
		"messages": [
			{"role": "user", "content": "Hello there."}
		]
	}`)

	pruned, model, stats, err := PruneAnthropicPayload(rawPayload, pipeline)
	if err != nil {
		t.Fatalf("failed to prune anthropic payload: %v", err)
	}

	if model != "claude-3-5-sonnet" {
		t.Fatalf("expected model claude-3-5-sonnet, got %s", model)
	}

	var res map[string]any
	if err := json.Unmarshal(pruned, &res); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if res["system"] != "You are a software engineer." {
		t.Fatalf("system prompt altered unexpectedly: %v", res["system"])
	}

	if stats.InboundTokens <= 0 {
		t.Fatalf("expected positive inbound tokens, got %d", stats.InboundTokens)
	}
}

func TestProxyServerStreamingIntegration(t *testing.T) {
	// 1. Mock upstream provider server emitting SSE chunks
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{
			"data: {\"choices\": [{\"delta\": {\"content\": \"Hello\"}}]}\n\n",
			"data: {\"choices\": [{\"delta\": {\"content\": \" world\"}}]}\n\n",
			"data: [DONE]\n\n",
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer mockUpstream.Close()

	// 2. Set up tokenprune config and in-memory store
	cfg := config.DefaultConfig()
	cfg.Server.UpstreamURL = mockUpstream.URL
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	proxyServer := NewServer(cfg, db)
	proxyHTTP := httptest.NewServer(proxyServer.Handler())
	defer proxyHTTP.Close()

	// 3. Make client request to proxy server
	clientReqBody := `{
		"model": "gpt-4o",
		"stream": true,
		"messages": [
			{"role": "user", "content": "Say hello world"}
		]
	}`

	req, err := http.NewRequest(http.MethodPost, proxyHTTP.URL+"/v1/chat/completions", bytes.NewBufferString(clientReqBody))
	if err != nil {
		t.Fatalf("failed to create client req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-123")
	req.Header.Set("X-Session-ID", "test-stream-sess")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read streamed body: %v", err)
	}

	bodyStr := string(bodyBytes)
	if !strings.Contains(bodyStr, "Hello") || !strings.Contains(bodyStr, "world") {
		t.Fatalf("streamed body missing expected tokens: %s", bodyStr)
	}

	// 4. Verify telemetry record was written into SQLite
	time.Sleep(100 * time.Millisecond) // allow async goroutine to persist
	sessions, err := db.ListSessions(10, 0)
	if err != nil {
		t.Fatalf("failed to query sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session record, got %d", len(sessions))
	}

	if sessions[0].SessionID != "test-stream-sess" {
		t.Fatalf("expected session ID test-stream-sess, got %s", sessions[0].SessionID)
	}
	if sessions[0].UpstreamTTFTMs <= 0 {
		t.Fatalf("expected positive TTFT, got %f", sessions[0].UpstreamTTFTMs)
	}
}
