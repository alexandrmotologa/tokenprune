package proxy

import (
	"encoding/json"

	"github.com/alexandrmotologa/tokenprune/internal/pruner"
)

// OpenAIRequest models the chat completions payload.
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Tools       []any           `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

// OpenAIMessage models a single message in OpenAI's API.
type OpenAIMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	Name       string            `json:"name,omitempty"`
	ToolCalls  []pruner.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

// ConvertOpenAIToUnified maps OpenAI messages to unified pruner messages.
func ConvertOpenAIToUnified(messages []OpenAIMessage) []pruner.Message {
	unified := make([]pruner.Message, len(messages))
	for i, m := range messages {
		unified[i] = pruner.Message{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}
	return unified
}

// ConvertUnifiedToOpenAI maps pruned messages back to OpenAI schema.
func ConvertUnifiedToOpenAI(messages []pruner.Message) []OpenAIMessage {
	openaiMsgs := make([]OpenAIMessage, len(messages))
	for i, m := range messages {
		openaiMsgs[i] = OpenAIMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}
	return openaiMsgs
}

// PruneOpenAIPayload parses and compresses an OpenAI chat completion body.
func PruneOpenAIPayload(body []byte, pipeline *pruner.Pipeline) ([]byte, string, pruner.PruningStats, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body, "", pruner.PruningStats{}, err
	}

	model, _ := req["model"].(string)
	rawMsgs, ok := req["messages"].([]any)
	if !ok {
		return body, model, pruner.PruningStats{}, nil
	}

	// Re-encode messages to parse into OpenAIMessage slice
	msgsBytes, err := json.Marshal(rawMsgs)
	if err != nil {
		return body, model, pruner.PruningStats{}, err
	}

	var openaiMsgs []OpenAIMessage
	if err := json.Unmarshal(msgsBytes, &openaiMsgs); err != nil {
		return body, model, pruner.PruningStats{}, err
	}

	unified := ConvertOpenAIToUnified(openaiMsgs)
	prunedUnified, stats := pipeline.Prune(unified, model)
	prunedOpenAI := ConvertUnifiedToOpenAI(prunedUnified)

	req["messages"] = prunedOpenAI
	prunedBody, err := json.Marshal(req)
	if err != nil {
		return body, model, stats, err
	}

	return prunedBody, model, stats, nil
}
