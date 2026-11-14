package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.Listen != ":8080" {
		t.Fatalf("expected listen :8080, got %s", cfg.Server.Listen)
	}
	if cfg.Server.TargetProvider != "anthropic" {
		t.Fatalf("expected target provider anthropic, got %s", cfg.Server.TargetProvider)
	}
	if cfg.Pruning.TargetCompressionRatio != 0.60 {
		t.Fatalf("expected target compression ratio 0.60, got %f", cfg.Pruning.TargetCompressionRatio)
	}
	if !cfg.Pruning.Tier1Syntactic.MinifyJSON {
		t.Fatal("expected MinifyJSON to be true by default")
	}
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_PORT", "9090")
	os.Setenv("TEST_PROVIDER", "openai")
	defer func() {
		os.Unsetenv("TEST_PORT")
		os.Unsetenv("TEST_PROVIDER")
	}()

	raw := "listen: :${TEST_PORT}\nprovider: ${TEST_PROVIDER}\nfallback: ${UNSET_VAR:-default_val}"
	expanded := ExpandEnv(raw)

	if !testing.Short() {
		if expected := "listen: :9090\nprovider: openai\nfallback: default_val"; expanded != expected {
			t.Fatalf("unexpected expansion:\nGot:\n%s\nExpected:\n%s", expanded, expected)
		}
	}
}

func TestLoadConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "tokenprune.yaml")

	content := `
version: "1"
server:
  listen: "3000"
  target_provider: "openai"
  sqlite_path: ":memory:"
pruning:
  target_compression_ratio: 0.75
  max_context_tokens: 32000
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Listen != ":3000" {
		t.Fatalf("expected listen :3000, got %s", cfg.Server.Listen)
	}
	if cfg.Server.TargetProvider != "openai" {
		t.Fatalf("expected provider openai, got %s", cfg.Server.TargetProvider)
	}
	if cfg.Server.UpstreamURL != "https://api.openai.com" {
		t.Fatalf("expected upstream URL https://api.openai.com, got %s", cfg.Server.UpstreamURL)
	}
	if cfg.Pruning.TargetCompressionRatio != 0.75 {
		t.Fatalf("expected compression ratio 0.75, got %f", cfg.Pruning.TargetCompressionRatio)
	}
}

func TestPriceForModel(t *testing.T) {
	cfg := DefaultConfig()
	priceSonnet := cfg.PriceForModel("claude-3-5-sonnet-20241022")
	if priceSonnet != 3.00 {
		t.Fatalf("expected 3.00 for claude-3-5-sonnet, got %f", priceSonnet)
	}

	priceMini := cfg.PriceForModel("gpt-4o-mini")
	if priceMini != 0.15 {
		t.Fatalf("expected 0.15 for gpt-4o-mini, got %f", priceMini)
	}

	priceUnknown := cfg.PriceForModel("unknown-model-xyz")
	if priceUnknown != cfg.Pricing.DefaultInputPerMillion {
		t.Fatalf("expected default %f for unknown, got %f", cfg.Pricing.DefaultInputPerMillion, priceUnknown)
	}
}
