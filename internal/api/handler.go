package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/metrics"
	"github.com/alexandrmotologa/tokenprune/internal/pruner"
	"github.com/alexandrmotologa/tokenprune/internal/proxy"
	"github.com/alexandrmotologa/tokenprune/internal/store"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

// Handler serves management and telemetry REST endpoints.
type Handler struct {
	cfg       *config.Config
	store     *store.DB
	pipeline  *pruner.Pipeline
	counter   *tokenizer.Counter
	startTime time.Time
}

// NewHandler constructs an API handler.
func NewHandler(cfg *config.Config, s *store.DB) *Handler {
	counter := tokenizer.NewCounter(10000)
	pipeline := pruner.NewPipeline(&cfg.Pruning, counter)

	return &Handler{
		cfg:       cfg,
		store:     s,
		pipeline:  pipeline,
		counter:   counter,
		startTime: time.Now(),
	}
}

// RegisterRoutes attaches API endpoints to the provided HTTP ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/api/sessions/", h.handleSessionDetail)
	mux.HandleFunc("/api/events", h.handleLiveEvents)
	mux.HandleFunc("/api/sandbox", h.handleSandbox)
	mux.HandleFunc("/api/config", h.handleConfig)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/metrics", metrics.Default.ServeHTTP)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uptime := time.Since(h.startTime).Seconds()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"uptime_seconds": uptime,
		"version":        "1.0.0",
		"provider":       h.cfg.Server.TargetProvider,
	})
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.store.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	offset := 0

	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if val, err := strconv.Atoi(lStr); err == nil && val > 0 {
			limit = val
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if val, err := strconv.Atoi(oStr); err == nil && val >= 0 {
			offset = val
		}
	}

	sessions, err := h.store.ListSessions(limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	if sessions == nil {
		sessions = []store.SessionRecord{}
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (h *Handler) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}

	session, err := h.store.GetSession(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *Handler) handleLiveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch, unsubscribe := h.store.Subscribe()
	defer unsubscribe()

	// Initial ping to acknowledge connection
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case record, ok := <-ch:
			if !ok {
				return
			}
			bytesData, err := json.Marshal(record)
			if err == nil {
				_, _ = fmt.Fprintf(w, "event: session\ndata: %s\n\n", bytesData)
				flusher.Flush()
			}
		}
	}
}

// SandboxRequest defines the payload for testing compression interactively.
type SandboxRequest struct {
	Text        string  `json:"text"`
	Model       string  `json:"model"`
	Format      string  `json:"format"` // "raw", "openai", "anthropic"
	Tier1       bool    `json:"tier1"`
	Tier2       bool    `json:"tier2"`
	Tier3       bool    `json:"tier3"`
	TargetRatio float64 `json:"target_ratio"`
	MaxTokens   int     `json:"max_tokens"`
}

// SandboxResponse returns detailed results and diffs.
type SandboxResponse struct {
	OriginalText string              `json:"original_text"`
	PrunedText   string              `json:"pruned_text"`
	Stats        pruner.PruningStats `json:"stats"`
	USDSaved     float64             `json:"usd_saved"`
}

func (h *Handler) handleSandbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req SandboxRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json request", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		req.Model = "claude-3-5-sonnet"
	}

	// Build custom ad-hoc configuration based on request
	cfg := h.cfg.Pruning
	cfg.Tier1Syntactic.MinifyJSON = req.Tier1
	cfg.Tier1Syntactic.NormalizeWhitespace = req.Tier1
	cfg.Tier1Syntactic.StripHTMLComments = req.Tier1
	cfg.Tier1Syntactic.PruneTablePadding = req.Tier1
	cfg.Tier2ToolEviction.Enabled = req.Tier2
	cfg.Tier3SemanticDelta.Enabled = req.Tier3

	if req.TargetRatio > 0 {
		cfg.TargetCompressionRatio = req.TargetRatio
	}
	if req.MaxTokens > 0 {
		cfg.MaxContextTokens = req.MaxTokens
	}

	customPipeline := pruner.NewPipeline(&cfg, h.counter)

	var prunedStr string
	var stats pruner.PruningStats

	trimmed := strings.TrimSpace(req.Text)
	if req.Format == "openai" || (req.Format == "" && strings.Contains(trimmed, `"messages"`)) {
		prunedBytes, _, s, err := proxy.PruneOpenAIPayload([]byte(req.Text), customPipeline)
		if err == nil {
			prunedStr = string(prunedBytes)
			stats = s
		} else {
			// Fallback to raw text pruning
			prunedStr, stats = h.pruneRawText(req.Text, req.Model, customPipeline)
		}
	} else if req.Format == "anthropic" {
		prunedBytes, _, s, err := proxy.PruneAnthropicPayload([]byte(req.Text), customPipeline)
		if err == nil {
			prunedStr = string(prunedBytes)
			stats = s
		} else {
			prunedStr, stats = h.pruneRawText(req.Text, req.Model, customPipeline)
		}
	} else {
		prunedStr, stats = h.pruneRawText(req.Text, req.Model, customPipeline)
	}

	usdSaved := (float64(stats.TokensSaved) / 1000000.0) * h.cfg.PriceForModel(req.Model)

	resp := SandboxResponse{
		OriginalText: req.Text,
		PrunedText:   prunedStr,
		Stats:        stats,
		USDSaved:     usdSaved,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) pruneRawText(text, model string, pipe *pruner.Pipeline) (string, pruner.PruningStats) {
	msgs := []pruner.Message{
		{Role: "user", Content: text},
	}
	prunedMsgs, stats := pipe.Prune(msgs, model)
	if len(prunedMsgs) > 0 {
		return prunedMsgs[0].Content, stats
	}
	return text, stats
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	safeCfg := *h.cfg
	// Mask webhook url for security
	if safeCfg.Notifications.Slack.WebhookURL != "" {
		safeCfg.Notifications.Slack.WebhookURL = "********"
	}

	writeJSON(w, http.StatusOK, safeCfg)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
