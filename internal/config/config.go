package config

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config defines the global configuration for the tokenprune proxy and engine.
type Config struct {
	Version       string        `yaml:"version"`
	Server        ServerConfig  `yaml:"server"`
	Pruning       PruningConfig `yaml:"pruning"`
	Pricing       PricingConfig `yaml:"pricing"`
	Notifications Notifications `yaml:"notifications"`
}

type ServerConfig struct {
	Listen              string `yaml:"listen"`
	TargetProvider      string `yaml:"target_provider"`
	UpstreamURL         string `yaml:"upstream_url"`
	SQLitePath          string `yaml:"sqlite_path"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type PruningConfig struct {
	TargetCompressionRatio float64                  `yaml:"target_compression_ratio"`
	MaxContextTokens       int                      `yaml:"max_context_tokens"`
	Tier1Syntactic         Tier1SyntacticConfig     `yaml:"tier1_syntactic"`
	Tier2ToolEviction      Tier2ToolEvictionConfig  `yaml:"tier2_tool_eviction"`
	Tier3SemanticDelta     Tier3SemanticDeltaConfig `yaml:"tier3_semantic_delta"`
}

type Tier1SyntacticConfig struct {
	MinifyJSON          bool `yaml:"minify_json"`
	StripHTMLComments   bool `yaml:"strip_html_comments"`
	NormalizeWhitespace bool `yaml:"normalize_whitespace"`
	PruneTablePadding   bool `yaml:"prune_table_padding"`
}

type Tier2ToolEvictionConfig struct {
	Enabled               bool     `yaml:"enabled"`
	EvictSupersededReads  bool     `yaml:"evict_superseded_reads"`
	MaxToolOutputAgeTurns int      `yaml:"max_tool_output_age_turns"`
	ReceiptTemplate       string   `yaml:"receipt_template"`
	PreserveErrors        bool     `yaml:"preserve_errors"`
	ProtectedToolNames    []string `yaml:"protected_tool_names"`
}

type Tier3SemanticDeltaConfig struct {
	Enabled            bool     `yaml:"enabled"`
	CompressAfterTurns int      `yaml:"compress_after_turns"`
	KeepRecentTurns    int      `yaml:"keep_recent_turns"`
	ProtectedKeywords  []string `yaml:"protected_keywords"`
}

type PricingConfig struct {
	DefaultInputPerMillion float64            `yaml:"default_input_per_million"`
	ModelRates             map[string]float64 `yaml:"model_rates"`
}

type Notifications struct {
	Slack SlackConfig `yaml:"slack"`
}

type SlackConfig struct {
	WebhookURL                 string  `yaml:"webhook_url"`
	AlertOnSavingsMilestoneUSD float64 `yaml:"alert_on_savings_milestone_usd"`
}

// DefaultConfig returns the default production configuration.
func DefaultConfig() *Config {
	return &Config{
		Version: "1",
		Server: ServerConfig{
			Listen:              ":8080",
			TargetProvider:      "anthropic",
			UpstreamURL:         "",
			SQLitePath:          "./tokenprune.db",
			ReadTimeoutSeconds:  120,
			WriteTimeoutSeconds: 300,
		},
		Pruning: PruningConfig{
			TargetCompressionRatio: 0.60,
			MaxContextTokens:       40000,
			Tier1Syntactic: Tier1SyntacticConfig{
				MinifyJSON:          true,
				StripHTMLComments:   true,
				NormalizeWhitespace: true,
				PruneTablePadding:   true,
			},
			Tier2ToolEviction: Tier2ToolEvictionConfig{
				Enabled:               true,
				EvictSupersededReads:  true,
				MaxToolOutputAgeTurns: 3,
				ReceiptTemplate:       "[Receipt: Tool '%s' (id: %s) output condensed (%d lines, %d bytes). Superseded by subsequent turn.]",
				PreserveErrors:        true,
				ProtectedToolNames:    []string{"bash_critical", "deploy", "checkout"},
			},
			Tier3SemanticDelta: Tier3SemanticDeltaConfig{
				Enabled:            true,
				CompressAfterTurns: 6,
				KeepRecentTurns:    4,
				ProtectedKeywords:  []string{"CRITICAL_RULES", "SYSTEM_INVARIANT", "PINNED"},
			},
		},
		Pricing: PricingConfig{
			DefaultInputPerMillion: 3.00,
			ModelRates: map[string]float64{
				"claude-3-7-sonnet": 3.00,
				"claude-3-5-sonnet": 3.00,
				"claude-3-5-haiku":  0.80,
				"claude-3-opus":     15.00,
				"gpt-4o-mini":       0.15,
				"gpt-4o":            2.50,
				"o1":                15.00,
				"o3-mini":           1.10,
				"gemini-1.5-pro":    1.25,
				"gemini-1.5-flash":  0.075,
				"gemini-2.0-flash":  0.10,
			},
		},
	}
}

var envPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}`)

// ExpandEnv replaces ${VAR} or ${VAR:-default} patterns in the input string.
func ExpandEnv(input string) string {
	return envPattern.ReplaceAllStringFunc(input, func(m string) string {
		sub := envPattern.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		varName := sub[1]
		defaultVal := ""
		if len(sub) >= 3 {
			defaultVal = sub[2]
		}
		if val, ok := os.LookupEnv(varName); ok && val != "" {
			return val
		}
		return defaultVal
	})
}

// LoadConfig reads a YAML configuration file, expands environment variables,
// and applies default fallbacks for unconfigured fields.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		cfg.ValidateAndNormalize()
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	expanded := ExpandEnv(string(data))
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml configuration: %w", err)
	}

	cfg.ValidateAndNormalize()
	return cfg, nil
}

// ValidateAndNormalize adjusts configuration fields to safe defaults.
func (c *Config) ValidateAndNormalize() {
	if c.Server.Listen == "" {
		c.Server.Listen = ":8080"
	}
	if !strings.HasPrefix(c.Server.Listen, ":") && !strings.Contains(c.Server.Listen, ":") {
		c.Server.Listen = ":" + c.Server.Listen
	}

	c.Server.TargetProvider = strings.ToLower(strings.TrimSpace(c.Server.TargetProvider))
	if c.Server.TargetProvider == "" {
		c.Server.TargetProvider = "anthropic"
	}

	if c.Server.UpstreamURL == "" {
		switch c.Server.TargetProvider {
		case "openai":
			c.Server.UpstreamURL = "https://api.openai.com"
		case "anthropic":
			c.Server.UpstreamURL = "https://api.anthropic.com"
		case "gemini":
			c.Server.UpstreamURL = "https://generativelanguage.googleapis.com"
		default:
			c.Server.UpstreamURL = "https://api.anthropic.com"
		}
	}

	if c.Server.SQLitePath == "" {
		c.Server.SQLitePath = "./tokenprune.db"
	}

	if c.Pruning.Tier2ToolEviction.ReceiptTemplate == "" {
		c.Pruning.Tier2ToolEviction.ReceiptTemplate = "[Receipt: Tool '%s' (id: %s) output condensed (%d lines, %d bytes). Superseded by subsequent turn.]"
	}

	if c.Pruning.Tier2ToolEviction.MaxToolOutputAgeTurns <= 0 {
		c.Pruning.Tier2ToolEviction.MaxToolOutputAgeTurns = 3
	}

	if c.Pruning.Tier3SemanticDelta.CompressAfterTurns <= 0 {
		c.Pruning.Tier3SemanticDelta.CompressAfterTurns = 6
	}

	if c.Pruning.Tier3SemanticDelta.KeepRecentTurns <= 0 {
		c.Pruning.Tier3SemanticDelta.KeepRecentTurns = 4
	}

	if c.Pricing.DefaultInputPerMillion <= 0 {
		c.Pricing.DefaultInputPerMillion = 3.00
	}
}

// PriceForModel returns the price in USD per 1 million input tokens for a given model.
func (c *Config) PriceForModel(model string) float64 {
	model = strings.ToLower(strings.TrimSpace(model))

	// Check exact matches first
	for k, v := range c.Pricing.ModelRates {
		if model == strings.ToLower(k) {
			return v
		}
	}

	// Sort keys by descending length to match longest/most specific model names first
	type rateEntry struct {
		key   string
		price float64
	}
	var entries []rateEntry
	for k, v := range c.Pricing.ModelRates {
		entries = append(entries, rateEntry{key: strings.ToLower(k), price: v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].key) > len(entries[j].key)
	})

	for _, entry := range entries {
		if strings.Contains(model, entry.key) {
			return entry.price
		}
	}
	return c.Pricing.DefaultInputPerMillion
}
