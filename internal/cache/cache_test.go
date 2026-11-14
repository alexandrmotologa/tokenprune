package cache

import (
	"testing"
	"time"
)

func TestToolCache(t *testing.T) {
	tc := NewToolCache(100 * time.Millisecond)

	tc.Set("git", "status", "status", 10, 200, "[Receipt: git status 10 lines]")

	entry, ok := tc.Get("git", "status")
	if !ok {
		t.Fatal("expected entry to be found in cache")
	}
	if entry.Lines != 10 || entry.Bytes != 200 {
		t.Fatalf("unexpected entry values: %+v", entry)
	}

	// Wait for TTL expiration
	time.Sleep(150 * time.Millisecond)

	_, ok = tc.Get("git", "status")
	if ok {
		t.Fatal("expected entry to be expired")
	}
}
