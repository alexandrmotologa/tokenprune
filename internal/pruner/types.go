package pruner

// ContentBlock represents polymorphic message content (text, tool_use, tool_result).
type ContentBlock struct {
	Type      string `json:"type,omitempty"` // "text", "tool_use", "tool_result", "image"
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`          // ID for tool_use or tool_result
	ToolUseID string `json:"tool_use_id,omitempty"` // Anthropic tool_result reference
	Name      string `json:"name,omitempty"`        // Tool name
	Input     any    `json:"input,omitempty"`       // Tool input arguments
	Content   any    `json:"content,omitempty"`     // Tool result content (string or blocks)
	IsError   bool   `json:"is_error,omitempty"`
}

// GetID returns either ID or ToolUseID for tool referencing.
func (b *ContentBlock) GetID() string {
	if b.ID != "" {
		return b.ID
	}
	return b.ToolUseID
}

// ToolCall represents an OpenAI-style tool call.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Message is the unified internal representation of a conversation turn.
type Message struct {
	Role       string         `json:"role"` // "system", "user", "assistant", "tool"
	Content    string         `json:"content,omitempty"`
	Blocks     []ContentBlock `json:"blocks,omitempty"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

// DiffItem records a single text modification or eviction for visual diffing.
type DiffItem struct {
	Tier        int    `json:"tier"` // 1, 2, or 3
	Description string `json:"description"`
	Original    string `json:"original"`
	Pruned      string `json:"pruned"`
	TokensSaved int    `json:"tokens_saved"`
}

// PruningStats records metrics for a completed pruning execution.
type PruningStats struct {
	InboundTokens       int        `json:"inbound_tokens"`
	OutboundTokens      int        `json:"outbound_tokens"`
	TokensSaved         int        `json:"tokens_saved"`
	CompressionRatio    float64    `json:"compression_ratio"`
	Tier1TokensSaved    int        `json:"tier1_tokens_saved"`
	Tier2TokensSaved    int        `json:"tier2_tokens_saved"`
	Tier3TokensSaved    int        `json:"tier3_tokens_saved"`
	EvictedToolsCount   int        `json:"evicted_tools_count"`
	ConsolidatedTurns   int        `json:"consolidated_turns"`
	ProcessingLatencyMs float64    `json:"processing_latency_ms"`
	Diffs               []DiffItem `json:"diffs,omitempty"`
}
