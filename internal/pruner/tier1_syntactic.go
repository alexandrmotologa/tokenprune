package pruner

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/tokenprune/internal/config"
	"github.com/alexandrmotologa/tokenprune/internal/tokenizer"
)

var (
	htmlCommentRegex = regexp.MustCompile(`<!--[\s\S]*?-->`)
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
	tableRowRegex    = regexp.MustCompile(`^\|(.+)\|$`)
)

// Tier1Syntactic executes lossless minification on messages.
type Tier1Syntactic struct {
	cfg     *config.Tier1SyntacticConfig
	counter *tokenizer.Counter
}

// NewTier1Syntactic creates a Tier1 pruner.
func NewTier1Syntactic(cfg *config.Tier1SyntacticConfig, counter *tokenizer.Counter) *Tier1Syntactic {
	return &Tier1Syntactic{
		cfg:     cfg,
		counter: counter,
	}
}

// ProcessMessage applies Tier 1 minification to a single message and returns diffs.
func (t *Tier1Syntactic) ProcessMessage(msg *Message, model string) []DiffItem {
	var diffs []DiffItem

	// 1. Minify content string if present
	if msg.Content != "" {
		original := msg.Content
		pruned := t.cleanText(original)

		if pruned != original {
			origTokens := t.counter.Estimate(original, model)
			prunedTokens := t.counter.Estimate(pruned, model)
			saved := origTokens - prunedTokens
			if saved > 0 {
				diffs = append(diffs, DiffItem{
					Tier:        1,
					Description: "Syntactic minification (JSON/whitespace/markup)",
					Original:    original,
					Pruned:      pruned,
					TokensSaved: saved,
				})
				msg.Content = pruned
			}
		}
	}

	// 2. Minify polymorphic blocks
	for i := range msg.Blocks {
		block := &msg.Blocks[i]
		if block.Text != "" {
			orig := block.Text
			cleaned := t.cleanText(orig)
			if cleaned != orig {
				saved := t.counter.Estimate(orig, model) - t.counter.Estimate(cleaned, model)
				if saved > 0 {
					diffs = append(diffs, DiffItem{
						Tier:        1,
						Description: "Content block minification",
						Original:    orig,
						Pruned:      cleaned,
						TokensSaved: saved,
					})
					block.Text = cleaned
				}
			}
		}

		// Tool result content might be a string
		if block.Type == "tool_result" {
			if strContent, ok := block.Content.(string); ok && strContent != "" {
				cleaned := t.cleanText(strContent)
				if cleaned != strContent {
					saved := t.counter.Estimate(strContent, model) - t.counter.Estimate(cleaned, model)
					if saved > 0 {
						diffs = append(diffs, DiffItem{
							Tier:        1,
							Description: "Tool result block minification",
							Original:    strContent,
							Pruned:      cleaned,
							TokensSaved: saved,
						})
						block.Content = cleaned
					}
				}
			}
		}
	}

	// 3. Minify OpenAI tool calls arguments
	for i := range msg.ToolCalls {
		tc := &msg.ToolCalls[i]
		if t.cfg.MinifyJSON && tc.Function.Arguments != "" {
			if minified, ok := MinifyJSON(tc.Function.Arguments); ok && minified != tc.Function.Arguments {
				saved := t.counter.Estimate(tc.Function.Arguments, model) - t.counter.Estimate(minified, model)
				if saved > 0 {
					diffs = append(diffs, DiffItem{
						Tier:        1,
						Description: "Tool call JSON argument minification",
						Original:    tc.Function.Arguments,
						Pruned:      minified,
						TokensSaved: saved,
					})
					tc.Function.Arguments = minified
				}
			}
		}
	}

	return diffs
}

// cleanText applies all enabled Tier 1 text transformation passes.
func (t *Tier1Syntactic) cleanText(raw string) string {
	text := raw

	// If entire text is valid JSON, minify it directly
	if t.cfg.MinifyJSON {
		if minified, ok := MinifyJSON(text); ok && len(minified) < len(text) {
			text = minified
		}
	}

	// Strip HTML comments
	if t.cfg.StripHTMLComments {
		text = htmlCommentRegex.ReplaceAllString(text, "")
	}

	// Normalize whitespace while preserving fenced code blocks
	if t.cfg.NormalizeWhitespace || t.cfg.PruneTablePadding {
		text = t.processLinesPreservingCodeBlocks(text)
	}

	return strings.TrimSpace(text)
}

// MinifyJSON compresses JSON by removing non-significant whitespace.
func MinifyJSON(input string) (string, bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return input, false
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(trimmed)); err != nil {
		return input, false
	}

	compacted := buf.String()
	// Only consider minified if it actually reduced length
	if len(compacted) < len(input) {
		return compacted, true
	}
	return input, false
}

// processLinesPreservingCodeBlocks cleans whitespace outside of ``` code blocks.
func (t *Tier1Syntactic) processLinesPreservingCodeBlocks(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "```") {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			continue
		}

		if inCodeBlock {
			// Do not modify anything inside code blocks
			result = append(result, line)
			continue
		}

		// Outside code blocks:
		curr := line

		// Prune markdown table cell padding: |  col1  |   col2   | -> | col1 | col2 |
		if t.cfg.PruneTablePadding && tableRowRegex.MatchString(trimmedLine) {
			curr = pruneTableRow(trimmedLine)
		} else if t.cfg.NormalizeWhitespace {
			// Collapse excessive consecutive horizontal spaces
			curr = collapseHorizontalSpaces(curr)
		}

		result = append(result, curr)
	}

	joined := strings.Join(result, "\n")
	if t.cfg.NormalizeWhitespace {
		joined = multiNewlineRegex.ReplaceAllString(joined, "\n\n")
	}
	return joined
}

// collapseHorizontalSpaces reduces runs of 2+ spaces outside code blocks to a single space.
func collapseHorizontalSpaces(s string) string {
	if len(s) == 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	spaceCount := 0

	for _, r := range s {
		if r == ' ' || r == '\t' {
			spaceCount++
			if spaceCount == 1 {
				b.WriteRune(' ')
			}
		} else {
			spaceCount = 0
			b.WriteRune(r)
		}
	}
	return b.String()
}

// pruneTableRow compacts extra padding spaces between markdown table pipes.
func pruneTableRow(row string) string {
	parts := strings.Split(row, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, " | ")
}
