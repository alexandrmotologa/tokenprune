package pruner

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

// Tier3StateDelta consolidates older turns into state transition records.
type Tier3StateDelta struct {
	cfg     *config.Tier3SemanticDeltaConfig
	counter *tokenizer.Counter
}

// NewTier3StateDelta creates a Tier3 pruner.
func NewTier3StateDelta(cfg *config.Tier3SemanticDeltaConfig, counter *tokenizer.Counter) *Tier3StateDelta {
	return &Tier3StateDelta{
		cfg:     cfg,
		counter: counter,
	}
}

// ProcessMessages consolidates older turns while anchoring system prompts and recent context.
func (t *Tier3StateDelta) ProcessMessages(messages []Message, model string) ([]Message, []DiffItem, int) {
	if !t.cfg.Enabled || len(messages) <= t.cfg.CompressAfterTurns {
		return messages, nil, 0
	}

	totalTurns := len(messages)
	keepRecent := t.cfg.KeepRecentTurns
	if keepRecent >= totalTurns {
		return messages, nil, 0
	}

	cutoffIdx := totalTurns - keepRecent
	var diffs []DiffItem
	consolidatedCount := 0

	// We preserve:
	// 1. All system messages
	// 2. Initial user message (turn 0) if it establishes the main prompt/task
	// 3. Any turn containing protected keywords
	// 4. All turns after cutoffIdx (the recent active working window)

	var output []Message
	var turnsToConsolidate []Message
	var originalBlocks []string

	for i, msg := range messages {
		// Rule 1: System messages are always kept
		if msg.Role == "system" {
			output = append(output, msg)
			continue
		}

		// Rule 2: Keep recent turns intact
		if i >= cutoffIdx {
			output = append(output, msg)
			continue
		}

		// Rule 3: Keep initial user message (defines root task)
		if i == 0 || (i == 1 && len(messages) > 1 && messages[0].Role == "system") {
			output = append(output, msg)
			continue
		}

		// Rule 4: Check protected keywords
		if t.containsProtectedKeyword(msg) {
			output = append(output, msg)
			continue
		}

		// Eligible for state delta consolidation
		turnsToConsolidate = append(turnsToConsolidate, msg)
		originalBlocks = append(originalBlocks, fmt.Sprintf("[%s]: %s", msg.Role, extractTextContent(msg)))
	}

	if len(turnsToConsolidate) >= 2 {
		consolidatedCount = len(turnsToConsolidate)
		summary := t.generateStateDeltaSummary(turnsToConsolidate)

		ledgerMessage := Message{
			Role:    "user",
			Content: summary,
		}

		origRaw := strings.Join(originalBlocks, "\n---\n")
		origTokens := t.counter.Estimate(origRaw, model)
		summaryTokens := t.counter.Estimate(summary, model)
		saved := origTokens - summaryTokens

		if saved > 0 {
			diffs = append(diffs, DiffItem{
				Tier:        3,
				Description: fmt.Sprintf("State delta consolidation (%d turns)", consolidatedCount),
				Original:    origRaw,
				Pruned:      summary,
				TokensSaved: saved,
			})

			// Insert ledger message right before the recent window
			// Find insertion point in output
			insertPos := len(output) - keepRecent
			if insertPos < 0 {
				insertPos = 0
			}

			newOutput := make([]Message, 0, len(output)+1)
			newOutput = append(newOutput, output[:insertPos]...)
			newOutput = append(newOutput, ledgerMessage)
			newOutput = append(newOutput, output[insertPos:]...)
			output = newOutput
		} else {
			// If no savings, keep original messages
			output = messages
			consolidatedCount = 0
		}
	} else {
		output = messages
	}

	return output, diffs, consolidatedCount
}

func (t *Tier3StateDelta) containsProtectedKeyword(msg Message) bool {
	text := strings.ToLower(extractTextContent(msg))
	for _, kw := range t.cfg.ProtectedKeywords {
		if strings.Contains(text, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// generateStateDeltaSummary constructs a deterministic, compact record of historical events.
func (t *Tier3StateDelta) generateStateDeltaSummary(turns []Message) string {
	var actions []string

	for _, msg := range turns {
		// Track tool actions
		for _, tc := range msg.ToolCalls {
			actions = append(actions, fmt.Sprintf("Called %s(%s)", tc.Function.Name, summarizeArguments(tc.Function.Arguments)))
		}
		for _, b := range msg.Blocks {
			if b.Type == "tool_use" {
				actions = append(actions, fmt.Sprintf("Used %s", b.Name))
			}
		}

		// Track short assistant milestones
		if msg.Role == "assistant" && len(msg.Content) > 0 && len(msg.Content) < 200 {
			firstLine := strings.Split(msg.Content, "\n")[0]
			actions = append(actions, firstLine)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[Session State Delta: Condensed %d intermediate turns.\n", len(turns)))
	if len(actions) > 0 {
		sb.WriteString("Completed actions and context:\n")
		// Keep up to 6 key actions
		limit := len(actions)
		if limit > 6 {
			limit = 6
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("- %s\n", actions[i]))
		}
	}
	sb.WriteString("Proceeding with active task context.]")

	return sb.String()
}

func summarizeArguments(args string) string {
	trimmed := strings.TrimSpace(args)
	if len(trimmed) > 40 {
		return trimmed[:37] + "..."
	}
	return trimmed
}

func extractTextContent(msg Message) string {
	if msg.Content != "" {
		return msg.Content
	}
	var sb strings.Builder
	for _, b := range msg.Blocks {
		if b.Text != "" {
			sb.WriteString(b.Text)
			sb.WriteString(" ")
		}
	}
	return strings.TrimSpace(sb.String())
}
