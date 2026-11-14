package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/store"
)

func TestAPIEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// Seed test session
	rec := &store.SessionRecord{
		ID:                  "sess-test-1",
		SessionID:           "test-agent-session",
		CreatedAt:           time.Now().UTC(),
		Provider:            "anthropic",
		Model:               "claude-3-5-sonnet",
		InboundTokens:       5000,
		OutboundTokens:      2000,
		TokensSaved:         3000,
		CompressionRatio:    60.0,
		Tier1Saved:          1000,
		Tier2Saved:          2000,
		Tier3Saved:          0,
		EvictedToolsCount:   1,
		ConsolidatedTurns:   0,
		ProcessingLatencyMs: 2.1,
		UpstreamTTFTMs:      320.0,
		USDSaved:            0.009,
	}
	if err := db.Record(rec); err != nil {
		t.Fatalf("failed to seed record: %v", err)
	}

	handler := NewHandler(cfg, db)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 1. Test /health
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	mux.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", recHealth.Code)
	}

	// 2. Test /api/stats
	reqStats := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	recStats := httptest.NewRecorder()
	mux.ServeHTTP(recStats, reqStats)
	if recStats.Code != http.StatusOK {
		t.Fatalf("expected /api/stats 200, got %d", recStats.Code)
	}

	var stats store.GlobalStats
	if err := json.NewDecoder(recStats.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats json: %v", err)
	}
	if stats.TotalRequests != 1 || stats.TotalTokensSaved != 3000 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	// 3. Test /api/sessions
	reqSessions := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	recSessions := httptest.NewRecorder()
	mux.ServeHTTP(recSessions, reqSessions)
	if recSessions.Code != http.StatusOK {
		t.Fatalf("expected /api/sessions 200, got %d", recSessions.Code)
	}

	var sessions []store.SessionRecord
	if err := json.NewDecoder(recSessions.Body).Decode(&sessions); err != nil {
		t.Fatalf("failed to decode sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "sess-test-1" {
		t.Fatalf("unexpected sessions list: %+v", sessions)
	}

	// 4. Test /api/sessions/sess-test-1
	reqDetail := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-test-1", nil)
	recDetail := httptest.NewRecorder()
	mux.ServeHTTP(recDetail, reqDetail)
	if recDetail.Code != http.StatusOK {
		t.Fatalf("expected session detail 200, got %d", recDetail.Code)
	}

	// 5. Test /api/sandbox
	sandboxPayload := `{
		"text": "{\n  \"key\": \"value\",\n  \"number\": 123\n}\n<!-- comment -->",
		"model": "claude-3-5-sonnet",
		"format": "raw",
		"tier1": true,
		"tier2": true,
		"tier3": true
	}`
	reqSandbox := httptest.NewRequest(http.MethodPost, "/api/sandbox", bytes.NewBufferString(sandboxPayload))
	recSandbox := httptest.NewRecorder()
	mux.ServeHTTP(recSandbox, reqSandbox)
	if recSandbox.Code != http.StatusOK {
		t.Fatalf("expected /api/sandbox 200, got %d", recSandbox.Code)
	}

	var sandboxResp SandboxResponse
	if err := json.NewDecoder(recSandbox.Body).Decode(&sandboxResp); err != nil {
		t.Fatalf("failed to decode sandbox response: %v", err)
	}
	if sandboxResp.Stats.TokensSaved < 0 {
		t.Fatal("expected non-negative tokens saved in sandbox")
	}
}
