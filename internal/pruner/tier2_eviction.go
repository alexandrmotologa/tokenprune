package pruner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

// ToolExecution holds parsed metadata about a tool invocation.
type ToolExecution struct {
	ID        string
	ToolName  string
	Target    string // extracted file path, command, or query
	CallTurn  int
	ReplyTurn int
	IsError   bool
}

// Tier2ToolEviction tracks tool outputs and evicts superseded or stale outputs.
type Tier2ToolEviction struct {
	cfg     *config.Tier2ToolEvictionConfig
	counter *tokenizer.Counter
}

// NewTier2ToolEviction creates a Tier2 pruner.
func NewTier2ToolEviction(cfg *config.Tier2ToolEvictionConfig, counter *tokenizer.Counter) *Tier2ToolEviction {
	return &Tier2ToolEviction{
		cfg:     cfg,
		counter: counter,
	}
}

// ProcessMessages inspects conversation history and evicts eligible tool outputs.
func (t *Tier2ToolEviction) ProcessMessages(messages []Message, model string) ([]DiffItem, int) {
	if !t.cfg.Enabled || len(messages) == 0 {
		return nil, 0
	}

	var diffs []DiffItem
	evictedCount := 0

	// 1. Build pointer-based index of tool invocations
	executions := t.indexToolExecutions(messages)
	totalTurns := len(messages)

	// Set of tool IDs eligible for eviction
	eligibleIDs := make(map[string]bool)

	// Map to track the latest execution for each target (e.g. file path)
	targetLatestCall := make(map[string]string) // key: target -> latest tool ID

	// Traverse in reverse to identify latest usage of each resource
	for i := len(executions) - 1; i >= 0; i-- {
		exec := executions[i]
		if exec.Target != "" {
			if _, exists := targetLatestCall[exec.Target]; !exists {
				targetLatestCall[exec.Target] = exec.ID
			}
		}
	}

	for _, exec := range executions {
		// Protected tool names cannot be evicted
		if t.isToolProtected(exec.ToolName) {
			continue
		}

		// Errors are preserved if configured
		if exec.IsError && t.cfg.PreserveErrors {
			continue
		}

		ageTurns := totalTurns - exec.ReplyTurn

		// Case A: Superseded by a newer call to the same resource
		isSuperseded := false
		if exec.Target != "" {
			if latestID, ok := targetLatestCall[exec.Target]; ok && latestID != exec.ID {
				isSuperseded = true
			}
		}

		// Case B: Stale by age (more than MaxToolOutputAgeTurns turns ago)
		isStale := ageTurns > t.cfg.MaxToolOutputAgeTurns

		if (isSuperseded && t.cfg.EvictSupersededReads) || isStale {
			eligibleIDs[exec.ID] = true
		}
	}

	// 2. Perform eviction on eligible tool outputs
	for i := range messages {
		msg := &messages[i]

		// OpenAI tool message
		if msg.Role == "tool" && msg.ToolCallID != "" && eligibleIDs[msg.ToolCallID] {
			if len(msg.Content) > 120 { // only evict if content is large enough to benefit
				lines := countLines(msg.Content)
				bytesLen := len(msg.Content)
				receipt := fmt.Sprintf(t.cfg.ReceiptTemplate, msg.Name, msg.ToolCallID, lines, bytesLen)

				origTokens := t.counter.Estimate(msg.Content, model)
				receiptTokens := t.counter.Estimate(receipt, model)
				saved := origTokens - receiptTokens

				if saved > 0 {
					diffs = append(diffs, DiffItem{
						Tier:        2,
						Description: fmt.Sprintf("Tool eviction: %s (id: %s)", msg.Name, msg.ToolCallID),
						Original:    msg.Content,
						Pruned:      receipt,
						TokensSaved: saved,
					})
					msg.Content = receipt
					evictedCount++
				}
			}
		}

		// Anthropic tool_result block
		for bIdx := range msg.Blocks {
			block := &msg.Blocks[bIdx]
			blockID := block.GetID()
			if block.Type == "tool_result" && blockID != "" && eligibleIDs[blockID] {
				if strContent, ok := block.Content.(string); ok && len(strContent) > 120 {
					lines := countLines(strContent)
					bytesLen := len(strContent)
					receipt := fmt.Sprintf(t.cfg.ReceiptTemplate, block.Name, blockID, lines, bytesLen)

					origTokens := t.counter.Estimate(strContent, model)
					receiptTokens := t.counter.Estimate(receipt, model)
					saved := origTokens - receiptTokens

					if saved > 0 {
						diffs = append(diffs, DiffItem{
							Tier:        2,
							Description: fmt.Sprintf("Tool result eviction: %s (id: %s)", block.Name, blockID),
							Original:    strContent,
							Pruned:      receipt,
							TokensSaved: saved,
						})
						block.Content = receipt
						evictedCount++
					}
				}
			}
		}
	}

	return diffs, evictedCount
}

func (t *Tier2ToolEviction) isToolProtected(name string) bool {
	for _, p := range t.cfg.ProtectedToolNames {
		if strings.EqualFold(p, name) {
			return true
		}
	}
	return false
}

// indexToolExecutions scans conversation messages to build an index of all tool calls and responses.
func (t *Tier2ToolEviction) indexToolExecutions(messages []Message) []*ToolExecution {
	var executions []*ToolExecution
	execByID := make(map[string]*ToolExecution)

	for turnIdx, msg := range messages {
		// Index OpenAI tool calls from assistant messages
		for _, tc := range msg.ToolCalls {
			target := extractTargetFromJSON(tc.Function.Arguments)
			exec := &ToolExecution{
				ID:       tc.ID,
				ToolName: tc.Function.Name,
				Target:   target,
				CallTurn: turnIdx,
			}
			execByID[tc.ID] = exec
			executions = append(executions, exec)
		}

		// Index Anthropic tool_use blocks
		for _, block := range msg.Blocks {
			if block.Type == "tool_use" {
				target := extractTargetFromAny(block.Input)
				exec := &ToolExecution{
					ID:       block.GetID(),
					ToolName: block.Name,
					Target:   target,
					CallTurn: turnIdx,
				}
				execByID[block.GetID()] = exec
				executions = append(executions, exec)
			}
		}

		// Match OpenAI tool responses
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if exec, ok := execByID[msg.ToolCallID]; ok {
				exec.ReplyTurn = turnIdx
				exec.IsError = detectErrorInOutput(msg.Content)
				if msg.Name == "" {
					msg.Name = exec.ToolName
				}
			}
		}

		// Match Anthropic tool_result blocks
		for bIdx := range msg.Blocks {
			block := &msg.Blocks[bIdx]
			blockID := block.GetID()
			if block.Type == "tool_result" && blockID != "" {
				if exec, ok := execByID[blockID]; ok {
					exec.ReplyTurn = turnIdx
					exec.IsError = block.IsError
					if !exec.IsError {
						if strContent, isStr := block.Content.(string); isStr {
							exec.IsError = detectErrorInOutput(strContent)
						}
					}
					if block.Name == "" {
						block.Name = exec.ToolName
					}
				}
			}
		}
	}

	return executions
}

// extractTargetFromJSON inspects tool arguments for keys like "path", "file", "command", "url".
func extractTargetFromJSON(args string) string {
	if args == "" {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(args), &data); err != nil {
		return ""
	}
	return extractTargetFromMap(data)
}

func extractTargetFromAny(val any) string {
	if val == nil {
		return ""
	}
	if m, ok := val.(map[string]any); ok {
		return extractTargetFromMap(m)
	}
	return ""
}

func extractTargetFromMap(m map[string]any) string {
	for _, key := range []string{"path", "file_path", "filename", "file", "command", "cmd", "url", "query"} {
		if v, ok := m[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func detectErrorInOutput(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "error:") ||
		strings.Contains(lower, "exception:") ||
		strings.Contains(lower, "traceback (most recent call last):") ||
		strings.Contains(lower, "exit status 1") ||
		strings.Contains(lower, "command failed") ||
		strings.Contains(lower, "fatal error")
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		lines++
	}
	return lines
}
