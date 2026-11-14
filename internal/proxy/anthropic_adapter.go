package proxy

import (
	"encoding/json"

	"github.com/alexandrmotologa/tokenprune/internal/pruner"
)

// AnthropicMessage represents a message in Anthropic's Messages API.
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content,omitempty"` // string or []pruner.ContentBlock
	Blocks  []any  `json:"blocks,omitempty"`  // alternative representation
}

// ConvertAnthropicToUnified maps Anthropic messages and system prompt to unified messages.
func ConvertAnthropicToUnified(system any, messages []AnthropicMessage) []pruner.Message {
	var unified []pruner.Message

	// Add system prompt as initial message if present
	if system != nil {
		if sysStr, ok := system.(string); ok && sysStr != "" {
			unified = append(unified, pruner.Message{
				Role:    "system",
				Content: sysStr,
			})
		}
	}

	for _, m := range messages {
		msg := pruner.Message{Role: m.Role}

		// Check blocks first
		rawBlocks := m.Blocks
		if len(rawBlocks) > 0 {
			blockBytes, err := json.Marshal(rawBlocks)
			if err == nil {
				var blocks []pruner.ContentBlock
				if err := json.Unmarshal(blockBytes, &blocks); err == nil {
					msg.Blocks = blocks
				}
			}
		}

		// Check content
		switch c := m.Content.(type) {
		case string:
			msg.Content = c
		case []any:
			blockBytes, err := json.Marshal(c)
			if err == nil {
				var blocks []pruner.ContentBlock
				if err := json.Unmarshal(blockBytes, &blocks); err == nil {
					msg.Blocks = append(msg.Blocks, blocks...)
				}
			}
		}

		unified = append(unified, msg)
	}

	return unified
}

// ConvertUnifiedToAnthropic reconstructs Anthropic messages and extracts system prompt.
func ConvertUnifiedToAnthropic(messages []pruner.Message) (any, []AnthropicMessage) {
	var system any
	var anthropicMsgs []AnthropicMessage

	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}

		aMsg := AnthropicMessage{Role: m.Role}
		if len(m.Blocks) > 0 {
			aMsg.Content = m.Blocks
		} else {
			aMsg.Content = m.Content
		}
		anthropicMsgs = append(anthropicMsgs, aMsg)
	}

	return system, anthropicMsgs
}

// PruneAnthropicPayload parses and compresses an Anthropic messages body.
func PruneAnthropicPayload(body []byte, pipeline *pruner.Pipeline) ([]byte, string, pruner.PruningStats, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body, "", pruner.PruningStats{}, err
	}

	model, _ := req["model"].(string)
	system := req["system"]

	rawMsgs, ok := req["messages"].([]any)
	if !ok {
		return body, model, pruner.PruningStats{}, nil
	}

	msgsBytes, err := json.Marshal(rawMsgs)
	if err != nil {
		return body, model, pruner.PruningStats{}, err
	}

	var anthropicMsgs []AnthropicMessage
	if err := json.Unmarshal(msgsBytes, &anthropicMsgs); err != nil {
		return body, model, pruner.PruningStats{}, err
	}

	unified := ConvertAnthropicToUnified(system, anthropicMsgs)
	prunedUnified, stats := pipeline.Prune(unified, model)
	newSystem, newAnthropicMsgs := ConvertUnifiedToAnthropic(prunedUnified)

	if newSystem != nil {
		req["system"] = newSystem
	}
	req["messages"] = newAnthropicMsgs

	prunedBody, err := json.Marshal(req)
	if err != nil {
		return body, model, stats, err
	}

	return prunedBody, model, stats, nil
}
