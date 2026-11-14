import React, { useEffect, useState } from 'react';
import { RuntimeConfig } from '../types';
import { Settings as SettingsIcon, Terminal, Database, Sliders } from 'lucide-react';

export const Settings: React.FC = () => {
  const [config, setConfig] = useState<RuntimeConfig | null>(null);

  useEffect(() => {
    fetch('/api/config')
      .then(res => res.json())
      .then(data => setConfig(data))
      .catch(err => console.error('Failed to load config', err));
  }, []);

  return (
    <div className="flex flex-col gap-6 max-w-5xl mx-auto">
      <div className="bg-slate-900/60 p-6 rounded-2xl border border-slate-800 shadow-xl backdrop-blur">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <SettingsIcon className="w-5 h-5 text-brand-400" />
          System Configuration & Client Integration
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Active runtime parameters and reverse proxy routing instructions
        </p>
      </div>

      {/* Integration Guide */}
      <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur">
        <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2 mb-3">
          <Terminal className="w-4 h-4 text-brand-400" />
          Agent Reverse Proxy Quickstart
        </h3>
        <p className="text-xs text-slate-400 mb-4">
          Point your agent framework to tokenprune. The proxy preserves all upstream headers and authentication tokens.
        </p>

        <div className="flex flex-col gap-3 font-mono text-xs">
          {/* Claude Code */}
          <div className="bg-slate-950 p-3.5 rounded-xl border border-slate-800/80">
            <div className="text-[11px] font-sans font-semibold text-slate-400 mb-1">
              Claude Code / Anthropic SDK
            </div>
            <code className="text-emerald-400 select-all">
              export ANTHROPIC_BASE_URL="http://localhost:8080"
            </code>
          </div>

          {/* OpenAI / Aider / Cursor */}
          <div className="bg-slate-950 p-3.5 rounded-xl border border-slate-800/80">
            <div className="text-[11px] font-sans font-semibold text-slate-400 mb-1">
              Aider / Cursor / OpenAI SDK
            </div>
            <code className="text-emerald-400 select-all">
              export OPENAI_BASE_URL="http://localhost:8080/v1"
            </code>
          </div>

          {/* Direct curl */}
          <div className="bg-slate-950 p-3.5 rounded-xl border border-slate-800/80">
            <div className="text-[11px] font-sans font-semibold text-slate-400 mb-1">
              Direct Streaming Test (cURL)
            </div>
            <code className="text-slate-300 select-all whitespace-pre-wrap">
              curl -N http://localhost:8080/v1/chat/completions \<br />
              &nbsp;&nbsp;-H "Content-Type: application/json" \<br />
              &nbsp;&nbsp;-H "Authorization: Bearer $OPENAI_API_KEY" \<br />
              &nbsp;&nbsp;-d '{`{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"Hello!"}]}`}'
            </code>
          </div>
        </div>
      </div>

      {/* Runtime Configuration Display */}
      {config && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Server & DB */}
          <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2 mb-4">
              <Database className="w-4 h-4 text-blue-400" />
              Network & Storage
            </h3>
            <div className="flex flex-col gap-2.5 text-xs">
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Proxy Listen Address</span>
                <span className="font-mono text-slate-200">{config.server.listen}</span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Target Provider</span>
                <span className="font-mono text-brand-400 uppercase font-semibold">
                  {config.server.target_provider}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Upstream URL</span>
                <span className="font-mono text-slate-200">{config.server.upstream_url}</span>
              </div>
              <div className="flex justify-between py-1.5">
                <span className="text-slate-400">SQLite Engine</span>
                <span className="font-mono text-slate-200">{config.server.sqlite_path}</span>
              </div>
            </div>
          </div>

          {/* Pruning Spec */}
          <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2 mb-4">
              <Sliders className="w-4 h-4 text-purple-400" />
              Pipeline Parameters
            </h3>
            <div className="flex flex-col gap-2.5 text-xs">
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Target Compression</span>
                <span className="font-mono text-slate-200">
                  {Math.round(config.pruning.target_compression_ratio * 100)}%
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Max Context Window</span>
                <span className="font-mono text-slate-200">
                  {config.pruning.max_context_tokens.toLocaleString()} tokens
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-slate-800">
                <span className="text-slate-400">Tier 2 Tool Output Max Age</span>
                <span className="font-mono text-slate-200">
                  {config.pruning.tier2_tool_eviction.max_tool_output_age_turns} turns
                </span>
              </div>
              <div className="flex justify-between py-1.5">
                <span className="text-slate-400">Tier 3 Delta Consolidation</span>
                <span className="font-mono text-slate-200">
                  After {config.pruning.tier3_semantic_delta.compress_after_turns} turns
                </span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
