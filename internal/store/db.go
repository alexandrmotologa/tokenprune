package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection and event subscription hub.
type DB struct {
	db       *sql.DB
	hubMu    sync.RWMutex
	subs     map[chan SessionRecord]struct{}
	closed   bool
	closeMu  sync.Mutex
}

// SessionRecord holds telemetry information for a single pruned request.
type SessionRecord struct {
	ID                  string    `json:"id"`
	SessionID           string    `json:"session_id"`
	CreatedAt           time.Time `json:"created_at"`
	Provider            string    `json:"provider"`
	Model               string    `json:"model"`
	InboundTokens       int       `json:"inbound_tokens"`
	OutboundTokens      int       `json:"outbound_tokens"`
	TokensSaved         int       `json:"tokens_saved"`
	CompressionRatio    float64   `json:"compression_ratio"`
	Tier1Saved          int       `json:"tier1_saved"`
	Tier2Saved          int       `json:"tier2_saved"`
	Tier3Saved          int       `json:"tier3_saved"`
	EvictedToolsCount   int       `json:"evicted_tools_count"`
	ConsolidatedTurns   int       `json:"consolidated_turns"`
	ProcessingLatencyMs float64   `json:"processing_latency_ms"`
	UpstreamTTFTMs      float64   `json:"upstream_ttft_ms"`
	USDSaved            float64   `json:"usd_saved"`
	RawPreview          string    `json:"raw_preview,omitempty"`
	PrunedPreview       string    `json:"pruned_preview,omitempty"`
}

// GlobalStats summarizes cumulative savings and performance metrics.
type GlobalStats struct {
	TotalRequests       int64   `json:"total_requests"`
	TotalInboundTokens  int64   `json:"total_inbound_tokens"`
	TotalOutboundTokens int64   `json:"total_outbound_tokens"`
	TotalTokensSaved    int64   `json:"total_tokens_saved"`
	TotalUSDSaved       float64 `json:"total_usd_saved"`
	AvgCompressionRatio float64 `json:"avg_compression_ratio"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	AvgTTFTMs           float64 `json:"avg_ttft_ms"`
}

// Open initializes SQLite and executes migration schema.
func Open(path string) (*DB, error) {
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create db directory: %w", err)
			}
		}
	}

	dsn := path
	if path != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", path)
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	sqlDB.SetMaxOpenConns(1) // SQLite operates best with a single writer connection

	store := &DB{
		db:   sqlDB,
		subs: make(map[chan SessionRecord]struct{}),
	}

	if err := store.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to run database migration: %w", err)
	}

	return store, nil
}

func (s *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		inbound_tokens INTEGER NOT NULL,
		outbound_tokens INTEGER NOT NULL,
		tokens_saved INTEGER NOT NULL,
		compression_ratio REAL NOT NULL,
		tier1_saved INTEGER NOT NULL,
		tier2_saved INTEGER NOT NULL,
		tier3_saved INTEGER NOT NULL,
		evicted_tools_count INTEGER NOT NULL,
		consolidated_turns INTEGER NOT NULL,
		processing_latency_ms REAL NOT NULL,
		upstream_ttft_ms REAL NOT NULL,
		usd_saved REAL NOT NULL,
		raw_preview TEXT,
		pruned_preview TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Record inserts a session record and broadcasts it to live subscribers.
func (s *DB) Record(rec *SessionRecord) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO sessions (
		id, session_id, created_at, provider, model,
		inbound_tokens, outbound_tokens, tokens_saved, compression_ratio,
		tier1_saved, tier2_saved, tier3_saved,
		evicted_tools_count, consolidated_turns,
		processing_latency_ms, upstream_ttft_ms, usd_saved,
		raw_preview, pruned_preview
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		rec.ID, rec.SessionID, rec.CreatedAt, rec.Provider, rec.Model,
		rec.InboundTokens, rec.OutboundTokens, rec.TokensSaved, rec.CompressionRatio,
		rec.Tier1Saved, rec.Tier2Saved, rec.Tier3Saved,
		rec.EvictedToolsCount, rec.ConsolidatedTurns,
		rec.ProcessingLatencyMs, rec.UpstreamTTFTMs, rec.USDSaved,
		rec.RawPreview, rec.PrunedPreview,
	)
	if err != nil {
		return fmt.Errorf("failed to insert session record: %w", err)
	}

	s.broadcast(*rec)
	return nil
}

// GetStats returns aggregated platform metrics.
func (s *DB) GetStats() (*GlobalStats, error) {
	query := `
	SELECT 
		COUNT(*),
		COALESCE(SUM(inbound_tokens), 0),
		COALESCE(SUM(outbound_tokens), 0),
		COALESCE(SUM(tokens_saved), 0),
		COALESCE(SUM(usd_saved), 0.0),
		COALESCE(AVG(compression_ratio), 0.0),
		COALESCE(AVG(processing_latency_ms), 0.0),
		COALESCE(AVG(upstream_ttft_ms), 0.0)
	FROM sessions
	`
	row := s.db.QueryRow(query)
	var stats GlobalStats
	err := row.Scan(
		&stats.TotalRequests,
		&stats.TotalInboundTokens,
		&stats.TotalOutboundTokens,
		&stats.TotalTokensSaved,
		&stats.TotalUSDSaved,
		&stats.AvgCompressionRatio,
		&stats.AvgLatencyMs,
		&stats.AvgTTFTMs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	return &stats, nil
}

// ListSessions returns recent sessions ordered by creation time descending.
func (s *DB) ListSessions(limit, offset int) ([]SessionRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
	SELECT 
		id, session_id, created_at, provider, model,
		inbound_tokens, outbound_tokens, tokens_saved, compression_ratio,
		tier1_saved, tier2_saved, tier3_saved,
		evicted_tools_count, consolidated_turns,
		processing_latency_ms, upstream_ttft_ms, usd_saved,
		raw_preview, pruned_preview
	FROM sessions
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?
	`
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var list []SessionRecord
	for rows.Next() {
		var r SessionRecord
		err := rows.Scan(
			&r.ID, &r.SessionID, &r.CreatedAt, &r.Provider, &r.Model,
			&r.InboundTokens, &r.OutboundTokens, &r.TokensSaved, &r.CompressionRatio,
			&r.Tier1Saved, &r.Tier2Saved, &r.Tier3Saved,
			&r.EvictedToolsCount, &r.ConsolidatedTurns,
			&r.ProcessingLatencyMs, &r.UpstreamTTFTMs, &r.USDSaved,
			&r.RawPreview, &r.PrunedPreview,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		list = append(list, r)
	}
	return list, nil
}

// GetSession returns a single session by ID.
func (s *DB) GetSession(id string) (*SessionRecord, error) {
	query := `
	SELECT 
		id, session_id, created_at, provider, model,
		inbound_tokens, outbound_tokens, tokens_saved, compression_ratio,
		tier1_saved, tier2_saved, tier3_saved,
		evicted_tools_count, consolidated_turns,
		processing_latency_ms, upstream_ttft_ms, usd_saved,
		raw_preview, pruned_preview
	FROM sessions
	WHERE id = ?
	`
	row := s.db.QueryRow(query, id)
	var r SessionRecord
	err := row.Scan(
		&r.ID, &r.SessionID, &r.CreatedAt, &r.Provider, &r.Model,
		&r.InboundTokens, &r.OutboundTokens, &r.TokensSaved, &r.CompressionRatio,
		&r.Tier1Saved, &r.Tier2Saved, &r.Tier3Saved,
		&r.EvictedToolsCount, &r.ConsolidatedTurns,
		&r.ProcessingLatencyMs, &r.UpstreamTTFTMs, &r.USDSaved,
		&r.RawPreview, &r.PrunedPreview,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}
	return &r, nil
}

// Subscribe returns a channel of live session records and an unsubscribe callback.
func (s *DB) Subscribe() (<-chan SessionRecord, func()) {
	s.hubMu.Lock()
	defer s.hubMu.Unlock()

	ch := make(chan SessionRecord, 100)
	s.subs[ch] = struct{}{}

	unsubscribe := func() {
		s.hubMu.Lock()
		defer s.hubMu.Unlock()
		if _, exists := s.subs[ch]; exists {
			delete(s.subs, ch)
			close(ch)
		}
	}

	return ch, unsubscribe
}

func (s *DB) broadcast(rec SessionRecord) {
	s.hubMu.RLock()
	defer s.hubMu.RUnlock()

	for ch := range s.subs {
		select {
		case ch <- rec:
		default:
			// Non-blocking drop if consumer buffer is full
		}
	}
}

// Close closes the underlying database.
func (s *DB) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.hubMu.Lock()
	for ch := range s.subs {
		close(ch)
	}
	s.subs = nil
	s.hubMu.Unlock()

	return s.db.Close()
}
