export interface SessionRecord {
  id: string;
  session_id: string;
  created_at: string;
  provider: string;
  model: string;
  inbound_tokens: number;
  outbound_tokens: number;
  tokens_saved: number;
  compression_ratio: number;
  tier1_saved: number;
  tier2_saved: number;
  tier3_saved: number;
  evicted_tools_count: number;
  consolidated_turns: number;
  processing_latency_ms: number;
  upstream_ttft_ms: number;
  usd_saved: number;
  raw_preview?: string;
  pruned_preview?: string;
}

export interface GlobalStats {
  total_requests: number;
  total_inbound_tokens: number;
  total_outbound_tokens: number;
  total_tokens_saved: number;
  total_usd_saved: number;
  avg_compression_ratio: number;
  avg_latency_ms: number;
  avg_ttft_ms: number;
}

export interface DiffItem {
  tier: number;
  description: string;
  original: string;
  pruned: string;
  tokens_saved: number;
}

export interface PruningStats {
  inbound_tokens: number;
  outbound_tokens: number;
  tokens_saved: number;
  compression_ratio: number;
  tier1_tokens_saved: number;
  tier2_tokens_saved: number;
  tier3_tokens_saved: number;
  evicted_tools_count: number;
  consolidated_turns: number;
  processing_latency_ms: number;
  diffs?: DiffItem[];
}

export interface SandboxResponse {
  original_text: string;
  pruned_text: string;
  stats: PruningStats;
  usd_saved: number;
}

export interface RuntimeConfig {
  version: string;
  server: {
    listen: string;
    target_provider: string;
    upstream_url: string;
    sqlite_path: string;
  };
  pruning: {
    target_compression_ratio: number;
    max_context_tokens: number;
    tier1_syntactic: {
      minify_json: boolean;
      strip_html_comments: boolean;
      normalize_whitespace: boolean;
      prune_table_padding: boolean;
    };
    tier2_tool_eviction: {
      enabled: boolean;
      evict_superseded_reads: boolean;
      max_tool_output_age_turns: number;
      preserve_errors: boolean;
      protected_tool_names: string[];
    };
    tier3_semantic_delta: {
      enabled: boolean;
      compress_after_turns: number;
      keep_recent_turns: number;
      protected_keywords: string[];
    };
  };
  pricing: {
    default_input_per_million: number;
    model_rates: Record<string, number>;
  };
}
