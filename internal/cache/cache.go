package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// CachedToolOutput holds metadata and cached receipt data for idempotent tool runs.
type CachedToolOutput struct {
	ToolName  string
	Target    string
	Hash      string
	Lines     int
	Bytes     int
	Receipt   string
	CreatedAt time.Time
}

// ToolCache manages in-memory caching of idempotent tool results.
type ToolCache struct {
	mu      sync.RWMutex
	entries map[string]CachedToolOutput
	ttl     time.Duration
}

// NewToolCache creates a cache with a specific TTL.
func NewToolCache(ttl time.Duration) *ToolCache {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &ToolCache{
		entries: make(map[string]CachedToolOutput),
		ttl:     ttl,
	}
}

// DefaultToolCache is a shared instance.
var DefaultToolCache = NewToolCache(2 * time.Minute)

// HashKey computes a deterministic key from tool name and arguments.
func HashKey(toolName string, args string) string {
	h := sha256.Sum256([]byte(toolName + ":" + args))
	return hex.EncodeToString(h[:16])
}

// Get returns the cached tool execution if unexpired.
func (c *ToolCache) Get(toolName string, args string) (CachedToolOutput, bool) {
	key := HashKey(toolName, args)
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return CachedToolOutput{}, false
	}

	if time.Since(entry.CreatedAt) > c.ttl {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return CachedToolOutput{}, false
	}

	return entry, true
}

// Set stores a tool execution result.
func (c *ToolCache) Set(toolName string, args string, target string, lines int, bytesLen int, receipt string) {
	key := HashKey(toolName, args)
	entry := CachedToolOutput{
		ToolName:  toolName,
		Target:    target,
		Hash:      key,
		Lines:     lines,
		Bytes:     bytesLen,
		Receipt:   receipt,
		CreatedAt: time.Now(),
	}

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

// Purge removes expired entries.
func (c *ToolCache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.entries {
		if now.Sub(v.CreatedAt) > c.ttl {
			delete(c.entries, k)
		}
	}
}
