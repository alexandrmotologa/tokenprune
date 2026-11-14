package pruner

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

// Pipeline manages multi-tier semantic compression and budget enforcement.
type Pipeline struct {
	cfg     *config.PruningConfig
	counter *tokenizer.Counter
	tier1   *Tier1Syntactic
	tier2   *Tier2ToolEviction
	tier3   *Tier3StateDelta
}

// NewPipeline initializes a compression pipeline with all tiers.
func NewPipeline(cfg *config.PruningConfig, counter *tokenizer.Counter) *Pipeline {
	if counter == nil {
		counter = tokenizer.DefaultCounter
	}
	return &Pipeline{
		cfg:     cfg,
		counter: counter,
		tier1:   NewTier1Syntactic(&cfg.Tier1Syntactic, counter),
		tier2:   NewTier2ToolEviction(&cfg.Tier2ToolEviction, counter),
		tier3:   NewTier3StateDelta(&cfg.Tier3SemanticDelta, counter),
	}
}

// Prune executes adaptive multi-tier compression on the conversation messages.
func (p *Pipeline) Prune(messages []Message, model string) ([]Message, PruningStats) {
	start := time.Now()

	working := make([]Message, len(messages))
	copy(working, messages)

	inboundTokens := p.CountMessages(working, model)
	var allDiffs []DiffItem

	tier1Saved := 0
	tier2Saved := 0
	tier3Saved := 0
	evictedTools := 0
	consolidatedTurns := 0

	// Step 1: Tier 1 Syntactic Minification
	for i := range working {
		diffs := p.tier1.ProcessMessage(&working[i], model)
		for _, d := range diffs {
			tier1Saved += d.TokensSaved
		}
		allDiffs = append(allDiffs, diffs...)
	}

	// Step 2: Tier 2 Tool Eviction
	if p.cfg.Tier2ToolEviction.Enabled {
		diffs, count := p.tier2.ProcessMessages(working, model)
		evictedTools = count
		for _, d := range diffs {
			tier2Saved += d.TokensSaved
		}
		allDiffs = append(allDiffs, diffs...)
	}

	// Step 3: Tier 3 State Delta Compression
	if p.cfg.Tier3SemanticDelta.Enabled && len(working) > p.cfg.Tier3SemanticDelta.CompressAfterTurns {
		var diffs []DiffItem
		working, diffs, consolidatedTurns = p.tier3.ProcessMessages(working, model)
		for _, d := range diffs {
			tier3Saved += d.TokensSaved
		}
		allDiffs = append(allDiffs, diffs...)
	}

	outboundTokens := p.CountMessages(working, model)
	tokensSaved := inboundTokens - outboundTokens
	if tokensSaved < 0 {
		tokensSaved = 0
	}

	actualRatio := 0.0
	if inboundTokens > 0 {
		actualRatio = (float64(tokensSaved) / float64(inboundTokens)) * 100.0
	}

	latencyMs := float64(time.Since(start).Nanoseconds()) / 1e6

	stats := PruningStats{
		InboundTokens:       inboundTokens,
		OutboundTokens:      outboundTokens,
		TokensSaved:         tokensSaved,
		CompressionRatio:    actualRatio,
		Tier1TokensSaved:    tier1Saved,
		Tier2TokensSaved:    tier2Saved,
		Tier3TokensSaved:    tier3Saved,
		EvictedToolsCount:   evictedTools,
		ConsolidatedTurns:   consolidatedTurns,
		ProcessingLatencyMs: latencyMs,
		Diffs:               allDiffs,
	}

	return working, stats
}

// CountMessages calculates total estimated tokens across all messages including envelope overhead.
func (p *Pipeline) CountMessages(messages []Message, model string) int {
	total := 0
	for _, m := range messages {
		total += tokenizer.MessageEnvelopeOverhead("anthropic")
		if m.Content != "" {
			total += p.counter.Estimate(m.Content, model)
		}
		for _, b := range m.Blocks {
			if b.Text != "" {
				total += p.counter.Estimate(b.Text, model)
			}
			if b.Name != "" {
				total += p.counter.Estimate(b.Name, model)
			}
			if b.Input != nil {
				if inBytes, err := json.Marshal(b.Input); err == nil {
					total += p.counter.Estimate(string(inBytes), model)
				}
			}
			if str, ok := b.Content.(string); ok && str != "" {
				total += p.counter.Estimate(str, model)
			} else if b.Content != nil {
				total += p.counter.Estimate(fmt.Sprintf("%v", b.Content), model)
			}
		}
		for _, tc := range m.ToolCalls {
			total += p.counter.Estimate(tc.Function.Name+tc.Function.Arguments, model)
		}
	}
	return total
}
