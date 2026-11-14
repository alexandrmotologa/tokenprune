package tokenizer

import (
	"strings"
	"testing"
)

func TestTokenEstimation(t *testing.T) {
	c := NewCounter(100)

	tests := []struct {
		name     string
		input    string
		minRange int
		maxRange int
	}{
		{
			name:     "empty string",
			input:    "",
			minRange: 0,
			maxRange: 0,
		},
		{
			name:     "short phrase",
			input:    "Hello world, how are you today?",
			minRange: 5,
			maxRange: 10,
		},
		{
			name:     "json payload",
			input:    `{"status": "success", "code": 200, "data": [{"id": 1, "name": "Item A"}, {"id": 2, "name": "Item B"}]}`,
			minRange: 20,
			maxRange: 45,
		},
		{
			name:     "long text",
			input:    strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20),
			minRange: 180,
			maxRange: 260,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := c.Estimate(tt.input, "gpt-4o")
			if count < tt.minRange || count > tt.maxRange {
				t.Fatalf("unexpected token count for %q: got %d, expected between %d and %d", tt.name, count, tt.minRange, tt.maxRange)
			}
		})
	}
}

func TestTokenCache(t *testing.T) {
	cache := NewTokenCache(3)

	cache.Set("key1", 10)
	cache.Set("key2", 20)
	cache.Set("key3", 30)

	if val, ok := cache.Get("key1"); !ok || val != 10 {
		t.Fatalf("expected key1 to be 10, got %d, %v", val, ok)
	}

	// Insert 4th item, should evict key1 (oldest in FIFO)
	cache.Set("key4", 40)

	if _, ok := cache.Get("key1"); ok {
		t.Fatal("expected key1 to be evicted")
	}

	if val, ok := cache.Get("key4"); !ok || val != 40 {
		t.Fatalf("expected key4 to be 40, got %d, %v", val, ok)
	}

	if cache.Size() != 3 {
		t.Fatalf("expected size 3, got %d", cache.Size())
	}
}

func BenchmarkTokenEstimation(b *testing.B) {
	c := NewCounter(1000)
	payload := `{"file": "/workspace/internal/proxy/server.go", "lines": 450, "content": "package proxy\nimport (\n\t\"net/http\"\n)\nfunc StartServer() error { return nil }\n"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Estimate(payload, "claude-3-5-sonnet")
	}
}
