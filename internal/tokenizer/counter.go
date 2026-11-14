package tokenizer

import (
	"strings"
	"unicode"
)

// Counter estimates token usage for prompts, messages, and raw strings.
type Counter struct {
	cache *TokenCache
}

// NewCounter creates a new Counter with a memoization cache.
func NewCounter(cacheSize int) *Counter {
	return &Counter{
		cache: NewTokenCache(cacheSize),
	}
}

// DefaultCounter is a shared instance for quick lookups.
var DefaultCounter = NewCounter(10000)

// Estimate returns the estimated token count for a string using tokenizer rules.
func (c *Counter) Estimate(text string, model string) int {
	if text == "" {
		return 0
	}

	if val, ok := c.cache.Get(text); ok {
		return val
	}

	count := calculateTokens(text, model)
	c.cache.Set(text, count)
	return count
}

// calculateTokens executes tokenization heuristics calibrated to OpenAI and Anthropic BPE patterns.
func calculateTokens(text string, model string) int {
	if len(text) == 0 {
		return 0
	}

	runes := []rune(text)
	n := len(runes)
	tokenCount := 0

	i := 0
	for i < n {
		r := runes[i]

		// Whitespace: newlines
		if r == '\n' {
			tokenCount++
			i++
			for i < n && runes[i] == '\n' {
				tokenCount++
				i++
			}
			continue
		}

		// Whitespace: spaces, tabs, carriage returns
		if unicode.IsSpace(r) {
			spaces := 0
			for i < n && unicode.IsSpace(runes[i]) && runes[i] != '\n' {
				if runes[i] == ' ' {
					spaces++
				} else if runes[i] == '\t' {
					spaces += 4
				}
				i++
			}
			if spaces > 1 {
				tokenCount += spaces / 4
			}
			continue
		}

		// Numbers: BPE chunks digits in groups of up to 3
		if unicode.IsDigit(r) {
			digits := 0
			for i < n && unicode.IsDigit(runes[i]) {
				digits++
				i++
			}
			tokenCount += (digits + 2) / 3
			continue
		}

		// Letters
		if unicode.IsLetter(r) {
			start := i
			for i < n && (unicode.IsLetter(runes[i]) || runes[i] == '\'') {
				i++
			}
			wordLen := i - start
			if wordLen <= 5 {
				tokenCount++
			} else {
				tokenCount += (wordLen + 3) / 4
			}
			continue
		}

		// Compound syntax/punctuation tokens common in BPE (JSON, code, XML)
		if i+1 < n {
			twoChar := string(runes[i : i+2])
			if twoChar == `":` || twoChar == `",` || twoChar == `{"` || twoChar == `}]` ||
				twoChar == `[{` || twoChar == `:=` || twoChar == `==` || twoChar == `!=` ||
				twoChar == `<>` || twoChar == `->` || twoChar == `/>` || twoChar == `</` ||
				twoChar == `//` || twoChar == `/*` || twoChar == `*/` || twoChar == `&&` ||
				twoChar == `||` {
				tokenCount++
				i += 2
				continue
			}
		}

		// Single punctuation and symbols
		start := i
		punctChar := r
		for i < n && runes[i] == punctChar && !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) && !unicode.IsSpace(runes[i]) {
			i++
		}
		punctLen := i - start
		if punctLen <= 1 {
			tokenCount++
			if start == i {
				i++
			}
		} else {
			tokenCount += (punctLen + 1) / 2
		}
	}

	if tokenCount < 1 && len(text) > 0 {
		tokenCount = 1
	}

	return tokenCount
}

// MessageEnvelopeOverhead returns the per-message wrapper tokens according to provider protocol.
func MessageEnvelopeOverhead(provider string) int {
	switch strings.ToLower(provider) {
	case "openai":
		return 3
	case "anthropic":
		return 2
	default:
		return 3
	}
}
