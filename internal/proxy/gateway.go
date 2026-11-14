package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/tokenprune/internal/pruner"
)

// OpenAIChunk represents a streaming chunk in OpenAI's API.
type OpenAIChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ConvertAnthropicResponseToOpenAI converts an Anthropic non-streaming JSON response to OpenAI format.
func ConvertAnthropicResponseToOpenAI(anthropicBody []byte, model string) ([]byte, error) {
	var resp map[string]any
	if err := json.Unmarshal(anthropicBody, &resp); err != nil {
		return anthropicBody, err
	}

	var textContent string
	if contentArr, ok := resp["content"].([]any); ok {
		for _, blockRaw := range contentArr {
			if blockMap, ok := blockRaw.(map[string]any); ok {
				if blockMap["type"] == "text" {
					if txt, ok := blockMap["text"].(string); ok {
						textContent += txt
					}
				}
			}
		}
	}

	openaiResp := map[string]any{
		"id":      fmt.Sprintf("chatcmpl_%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": textContent,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
	}

	return json.Marshal(openaiResp)
}

// TranslateAnthropicSSEToOpenAI takes a raw SSE line from Anthropic and formats it as an OpenAI SSE chunk.
func TranslateAnthropicSSEToOpenAI(sseLine string, model string, streamID string) string {
	trimmed := strings.TrimSpace(sseLine)
	if trimmed == "" || strings.HasPrefix(trimmed, "event:") {
		return ""
	}

	if !strings.HasPrefix(trimmed, "data:") {
		return ""
	}

	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if payload == "[DONE]" {
		return "data: [DONE]\n\n"
	}

	var eventData map[string]any
	if err := json.Unmarshal([]byte(payload), &eventData); err != nil {
		return ""
	}

	eventType, _ := eventData["type"].(string)

	if eventType == "content_block_delta" {
		if delta, ok := eventData["delta"].(map[string]any); ok {
			if deltaType, _ := delta["type"].(string); deltaType == "text_delta" {
				if text, ok := delta["text"].(string); ok {
					chunk := OpenAIChunk{
						ID:      streamID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   model,
					}
					chunk.Choices = append(chunk.Choices, struct {
						Index int `json:"index"`
						Delta struct {
							Role    string `json:"role,omitempty"`
							Content string `json:"content,omitempty"`
						} `json:"delta"`
						FinishReason *string `json:"finish_reason"`
					}{
						Index: 0,
						Delta: struct {
							Role    string `json:"role,omitempty"`
							Content string `json:"content,omitempty"`
						}{
							Content: text,
						},
						FinishReason: nil,
					})

					bytesData, _ := json.Marshal(chunk)
					return fmt.Sprintf("data: %s\n\n", bytesData)
				}
			}
		}
	} else if eventType == "message_stop" {
		return "data: [DONE]\n\n"
	}

	return ""
}

// TranslateOpenAIToAnthropicPayload transforms OpenAI request JSON into Anthropic request JSON.
func TranslateOpenAIToAnthropicPayload(body []byte, pipeline *pruner.Pipeline) ([]byte, string, pruner.PruningStats, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body, "", pruner.PruningStats{}, err
	}

	model, _ := req["model"].(string)
	rawMsgs, ok := req["messages"].([]any)
	if !ok {
		return body, model, pruner.PruningStats{}, nil
	}

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

	system, anthropicMsgs := ConvertUnifiedToAnthropic(prunedUnified)

	anthropicReq := map[string]any{
		"model":      model,
		"messages":   anthropicMsgs,
		"max_tokens": 4096,
	}
	if stream, ok := req["stream"].(bool); ok {
		anthropicReq["stream"] = stream
	}
	if system != nil {
		anthropicReq["system"] = system
	}

	prunedBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return body, model, stats, err
	}

	return prunedBody, model, stats, nil
}
