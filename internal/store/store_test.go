package store

import (
	"testing"
	"time"
)

func TestStoreOperations(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	// Test subscription
	ch, unsub := db.Subscribe()
	defer unsub()

	rec := &SessionRecord{
		ID:                  "test-rec-1",
		SessionID:           "sess-abc",
		CreatedAt:           time.Now().UTC(),
		Provider:            "anthropic",
		Model:               "claude-3-5-sonnet",
		InboundTokens:       10000,
		OutboundTokens:      3500,
		TokensSaved:         6500,
		CompressionRatio:    65.0,
		Tier1Saved:          1500,
		Tier2Saved:          3000,
		Tier3Saved:          2000,
		EvictedToolsCount:   2,
		ConsolidatedTurns:   4,
		ProcessingLatencyMs: 3.5,
		UpstreamTTFTMs:      450.0,
		USDSaved:            0.0195,
		RawPreview:          "raw prompt preview",
		PrunedPreview:       "pruned prompt preview",
	}

	if err := db.Record(rec); err != nil {
		t.Fatalf("failed to record session: %v", err)
	}

	// Verify subscriber received the broadcast
	select {
	case received := <-ch:
		if received.ID != rec.ID {
			t.Fatalf("expected id %s, got %s", rec.ID, received.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event broadcast")
	}

	// Verify GetSession
	fetched, err := db.GetSession("test-rec-1")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if fetched == nil || fetched.ID != "test-rec-1" {
		t.Fatalf("expected session test-rec-1, got %v", fetched)
	}

	// Verify ListSessions
	list, err := db.ListSessions(10, 0)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}

	// Verify GetStats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalRequests != 1 {
		t.Fatalf("expected 1 total request, got %d", stats.TotalRequests)
	}
	if stats.TotalTokensSaved != 6500 {
		t.Fatalf("expected 6500 tokens saved, got %d", stats.TotalTokensSaved)
	}
	if stats.AvgCompressionRatio != 65.0 {
		t.Fatalf("expected 65.0 avg compression ratio, got %f", stats.AvgCompressionRatio)
	}
}
