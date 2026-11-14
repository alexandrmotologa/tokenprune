package pruner

import (
	"fmt"
	"strings"
)

// ExtractInvariants finds invariant sentences or constraints from system and initial prompts.
func ExtractInvariants(messages []Message, keywords []string) string {
	if len(messages) == 0 {
		return ""
	}

	var candidates []string

	for i := 0; i < len(messages) && i < 3; i++ {
		msg := messages[i]
		if msg.Role != "system" && msg.Role != "user" {
			continue
		}

		text := extractTextContent(msg)
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) == 0 {
				continue
			}
			lower := strings.ToLower(trimmed)
			for _, kw := range keywords {
				if strings.Contains(lower, strings.ToLower(kw)) {
					candidates = append(candidates, trimmed)
					break
				}
			}
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Keep up to 3 most important invariants
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}

	return strings.Join(candidates, " | ")
}

// ReinforceAnchor injects an active constraint reminder into the penultimate position
// if conversation turn count exceeds the threshold.
func ReinforceAnchor(messages []Message, invariants string, threshold int) ([]Message, *DiffItem, bool) {
	if len(messages) < threshold || invariants == "" {
		return messages, nil, false
	}

	reminder := fmt.Sprintf("[Constraint Anchor: %s]", invariants)

	// Check if already anchored recently to avoid duplicate stacking
	lastIdx := len(messages) - 1
	if lastIdx > 0 && strings.Contains(extractTextContent(messages[lastIdx]), "Constraint Anchor:") {
		return messages, nil, false
	}
	if lastIdx > 1 && strings.Contains(extractTextContent(messages[lastIdx-1]), "Constraint Anchor:") {
		return messages, nil, false
	}

	// Attach anchor to the last user message or insert right before the last user turn
	updated := make([]Message, len(messages))
	copy(updated, messages)

	targetTurn := &updated[lastIdx]
	if targetTurn.Role == "user" && targetTurn.Content != "" {
		original := targetTurn.Content
		targetTurn.Content = reminder + "\n\n" + original
		diff := &DiffItem{
			Tier:        3,
			Description: "Context drift anchor reinforced",
			Original:    original,
			Pruned:      targetTurn.Content,
			TokensSaved: 0,
		}
		return updated, diff, true
	}

	return messages, nil, false
}
