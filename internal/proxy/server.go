package proxy

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/pruner"
	"github.com/alexandrmotologa/tokenprune/internal/store"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

// Server implements the streaming reverse proxy.
type Server struct {
	cfg      *config.Config
	pipeline *pruner.Pipeline
	store    *store.DB
	client   *http.Client
}

// NewServer creates a proxy server instance.
func NewServer(cfg *config.Config, s *store.DB) *Server {
	counter := tokenizer.NewCounter(20000)
	pipeline := pruner.NewPipeline(&cfg.Pruning, counter)

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	return &Server{
		cfg:      cfg,
		pipeline: pipeline,
		store:    s,
		client:   client,
	}
}

// Handler returns the HTTP handler for proxying requests.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.ServeHTTP)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS handling for browser clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, anthropic-version, X-Session-ID")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path

	// Route to pruning interceptor if chat completion or messages endpoint
	isOpenAI := strings.Contains(path, "/chat/completions")
	isAnthropic := strings.Contains(path, "/messages")

	if r.Method == http.MethodPost && (isOpenAI || isAnthropic) {
		s.handlePrunedRequest(w, r, isOpenAI)
		return
	}

	// Passthrough for all other endpoints (e.g. /v1/models, /health)
	s.handlePassthrough(w, r)
}

func (s *Server) handlePrunedRequest(w http.ResponseWriter, r *http.Request, isOpenAI bool) {
	startTime := time.Now()
	reqID := generateID("req")
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = generateID("sess")
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	var prunedBody []byte
	var model string
	var stats pruner.PruningStats
	var pruneErr error

	if isOpenAI {
		prunedBody, model, stats, pruneErr = PruneOpenAIPayload(rawBody, s.pipeline)
	} else {
		prunedBody, model, stats, pruneErr = PruneAnthropicPayload(rawBody, s.pipeline)
	}

	if pruneErr != nil {
		// On unexpected parsing error, fallback to raw body to never break upstream agent
		prunedBody = rawBody
	}

	upstreamURL := strings.TrimRight(s.cfg.Server.UpstreamURL, "/") + r.URL.RequestURI()
	upReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(prunedBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to construct upstream request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy upstream request headers
	for k, vv := range r.Header {
		for _, v := range vv {
			upReq.Header.Add(k, v)
		}
	}
	upReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(prunedBody)))
	upReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

	resp, err := s.client.Do(upReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	flusher, isFlusher := w.(http.Flusher)
	isSSE := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	var ttftMs float64
	firstByteRecorded := false
	buf := make([]byte, 4096)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if !firstByteRecorded {
				ttftMs = float64(time.Since(startTime).Nanoseconds()) / 1e6
				firstByteRecorded = true
			}
			_, _ = w.Write(buf[:n])
			if isSSE && isFlusher {
				flusher.Flush()
			}
		}
		if readErr != nil {
			break
		}
	}

	if !firstByteRecorded {
		ttftMs = float64(time.Since(startTime).Nanoseconds()) / 1e6
	}

	// Calculate and record telemetry asynchronously to not block response
	go func() {
		usdSaved := (float64(stats.TokensSaved) / 1000000.0) * s.cfg.PriceForModel(model)

		rawPreview := previewString(string(rawBody), 250)
		prunedPreview := previewString(string(prunedBody), 250)

		provider := s.cfg.Server.TargetProvider
		if isOpenAI {
			provider = "openai"
		} else {
			provider = "anthropic"
		}

		record := &store.SessionRecord{
			ID:                  reqID,
			SessionID:           sessionID,
			CreatedAt:           time.Now().UTC(),
			Provider:            provider,
			Model:               model,
			InboundTokens:       stats.InboundTokens,
			OutboundTokens:      stats.OutboundTokens,
			TokensSaved:         stats.TokensSaved,
			CompressionRatio:    stats.CompressionRatio,
			Tier1Saved:          stats.Tier1TokensSaved,
			Tier2Saved:          stats.Tier2TokensSaved,
			Tier3Saved:          stats.Tier3TokensSaved,
			EvictedToolsCount:   stats.EvictedToolsCount,
			ConsolidatedTurns:   stats.ConsolidatedTurns,
			ProcessingLatencyMs: stats.ProcessingLatencyMs,
			UpstreamTTFTMs:      ttftMs,
			USDSaved:            usdSaved,
			RawPreview:          rawPreview,
			PrunedPreview:       prunedPreview,
		}

		_ = s.store.Record(record)
	}()
}

func (s *Server) handlePassthrough(w http.ResponseWriter, r *http.Request) {
	upstreamURL := strings.TrimRight(s.cfg.Server.UpstreamURL, "/") + r.URL.RequestURI()
	upReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create passthrough request: %v", err), http.StatusInternalServerError)
		return
	}

	for k, vv := range r.Header {
		for _, v := range vv {
			upReq.Header.Add(k, v)
		}
	}

	resp, err := s.client.Do(upReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func previewString(s string, maxLen int) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) > maxLen {
		return trimmed[:maxLen] + "..."
	}
	return trimmed
}
