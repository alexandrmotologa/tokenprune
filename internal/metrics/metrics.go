package metrics

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/alexandrmotologa/tokenprune/internal/store"
)

// Collector tracks real-time Prometheus metrics.
type Collector struct {
	mu                sync.RWMutex
	requestsByModel   map[string]*int64
	tokensSavedByTier map[string]*int64
	usdSavedByModel   map[string]*float64
	inboundTokens     int64
	outboundTokens    int64
	totalTokensSaved  int64
	latencySumMs      float64
	latencyCount      int64
	ttftSumMs         float64
	ttftCount         int64
}

// NewCollector constructs a thread-safe metrics collector.
func NewCollector() *Collector {
	return &Collector{
		requestsByModel:   make(map[string]*int64),
		tokensSavedByTier: make(map[string]*int64),
		usdSavedByModel:   make(map[string]*float64),
	}
}

// Default is the singleton collector used across proxy and API.
var Default = NewCollector()

// ObserveSession updates Prometheus metrics with a completed session.
func (m *Collector) ObserveSession(rec *store.SessionRecord) {
	atomic.AddInt64(&m.inboundTokens, int64(rec.InboundTokens))
	atomic.AddInt64(&m.outboundTokens, int64(rec.OutboundTokens))
	atomic.AddInt64(&m.totalTokensSaved, int64(rec.TokensSaved))

	m.mu.Lock()
	defer m.mu.Unlock()

	// Requests by model
	key := fmt.Sprintf("%s|%s", rec.Provider, rec.Model)
	if _, ok := m.requestsByModel[key]; !ok {
		var count int64
		m.requestsByModel[key] = &count
	}
	*m.requestsByModel[key]++

	// Tier savings
	t1Key := "tier1"
	if _, ok := m.tokensSavedByTier[t1Key]; !ok {
		var v int64
		m.tokensSavedByTier[t1Key] = &v
	}
	*m.tokensSavedByTier[t1Key] += int64(rec.Tier1Saved)

	t2Key := "tier2"
	if _, ok := m.tokensSavedByTier[t2Key]; !ok {
		var v int64
		m.tokensSavedByTier[t2Key] = &v
	}
	*m.tokensSavedByTier[t2Key] += int64(rec.Tier2Saved)

	t3Key := "tier3"
	if _, ok := m.tokensSavedByTier[t3Key]; !ok {
		var v int64
		m.tokensSavedByTier[t3Key] = &v
	}
	*m.tokensSavedByTier[t3Key] += int64(rec.Tier3Saved)

	// USD savings
	if _, ok := m.usdSavedByModel[rec.Model]; !ok {
		var val float64
		m.usdSavedByModel[rec.Model] = &val
	}
	*m.usdSavedByModel[rec.Model] += rec.USDSaved

	// Latency
	m.latencySumMs += rec.ProcessingLatencyMs
	m.latencyCount++

	if rec.UpstreamTTFTMs > 0 {
		m.ttftSumMs += rec.UpstreamTTFTMs
		m.ttftCount++
	}
}

// ServeHTTP writes metrics in the standard Prometheus text exposition format.
func (m *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	var sb strings.Builder

	// Requests total
	sb.WriteString("# HELP tokenprune_requests_total Total number of proxy requests processed.\n")
	sb.WriteString("# TYPE tokenprune_requests_total counter\n")
	for key, countPtr := range m.requestsByModel {
		parts := strings.Split(key, "|")
		provider := parts[0]
		model := parts[1]
		sb.WriteString(fmt.Sprintf("tokenprune_requests_total{provider=\"%s\",model=\"%s\"} %d\n", provider, model, *countPtr))
	}

	// Inbound & Outbound tokens
	sb.WriteString("# HELP tokenprune_tokens_inbound_total Cumulative inbound prompt tokens.\n")
	sb.WriteString("# TYPE tokenprune_tokens_inbound_total counter\n")
	sb.WriteString(fmt.Sprintf("tokenprune_tokens_inbound_total %d\n", atomic.LoadInt64(&m.inboundTokens)))

	sb.WriteString("# HELP tokenprune_tokens_outbound_total Cumulative outbound pruned prompt tokens.\n")
	sb.WriteString("# TYPE tokenprune_tokens_outbound_total counter\n")
	sb.WriteString(fmt.Sprintf("tokenprune_tokens_outbound_total %d\n", atomic.LoadInt64(&m.outboundTokens)))

	sb.WriteString("# HELP tokenprune_tokens_saved_total Cumulative prompt tokens saved across tiers.\n")
	sb.WriteString("# TYPE tokenprune_tokens_saved_total counter\n")
	for tier, savedPtr := range m.tokensSavedByTier {
		sb.WriteString(fmt.Sprintf("tokenprune_tokens_saved_total{tier=\"%s\"} %d\n", tier, *savedPtr))
	}

	// USD saved
	sb.WriteString("# HELP tokenprune_cost_avoidance_usd_total Estimated API expense avoided in USD.\n")
	sb.WriteString("# TYPE tokenprune_cost_avoidance_usd_total counter\n")
	for model, usdPtr := range m.usdSavedByModel {
		sb.WriteString(fmt.Sprintf("tokenprune_cost_avoidance_usd_total{model=\"%s\"} %.6f\n", model, *usdPtr))
	}

	// Latency
	sb.WriteString("# HELP tokenprune_processing_latency_ms Summary of proxy pruning latency.\n")
	sb.WriteString("# TYPE tokenprune_processing_latency_ms summary\n")
	sb.WriteString(fmt.Sprintf("tokenprune_processing_latency_ms_sum %.4f\n", m.latencySumMs))
	sb.WriteString(fmt.Sprintf("tokenprune_processing_latency_ms_count %d\n", m.latencyCount))

	// Upstream TTFT
	sb.WriteString("# HELP tokenprune_upstream_ttft_ms Summary of upstream Time to First Token.\n")
	sb.WriteString("# TYPE tokenprune_upstream_ttft_ms summary\n")
	sb.WriteString(fmt.Sprintf("tokenprune_upstream_ttft_ms_sum %.2f\n", m.ttftSumMs))
	sb.WriteString(fmt.Sprintf("tokenprune_upstream_ttft_ms_count %d\n", m.ttftCount))

	_, _ = w.Write([]byte(sb.String()))
}
