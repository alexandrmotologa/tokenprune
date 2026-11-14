import React, { useState } from 'react';
import { SandboxResponse } from '../types';
import { DiffViewer } from './DiffViewer';
import { Play, Sparkles, RotateCcw } from 'lucide-react';

const SAMPLE_BLOATED_PAYLOAD = `{
  "model": "claude-3-5-sonnet",
  "messages": [
    {
      "role": "user",
      "content": "Please read the config file and inspect the project."
    },
    {
      "role": "assistant",
      "content": "Understood. Reading configuration file.",
      "blocks": [
        {
          "type": "tool_use",
          "id": "toolu_read_cfg_1",
          "name": "read_file",
          "input": {
            "path": "internal/config/config.go"
          }
        }
      ]
    },
    {
      "role": "user",
      "blocks": [
        {
          "type": "tool_result",
          "id": "toolu_read_cfg_1",
          "name": "read_file",
          "content": "package config\\n\\nimport (\\n\\t\\"fmt\\"\\n\\t\\"os\\"\\n\\t\\"regexp\\"\\n)\\n\\n// Configuration struct with redundant documentation lines.\\n// Line 1: Setting listen port\\n// Line 2: Setting upstream provider\\n// Line 3: Setting sqlite path\\n// Line 4: Setting tier1 options\\n// Line 5: Setting tier2 options\\n// Line 6: Setting tier3 options\\n// Line 7: Setting pricing options\\n// Line 8: Additional comment that inflates context window unnecessarily.\\n// Line 9: Repeated comment line.\\n// Line 10: Repeated comment line.\\n"
        }
      ]
    },
    {
      "role": "assistant",
      "content": "Config loaded. Now writing updated configuration.",
      "blocks": [
        {
          "type": "tool_use",
          "id": "toolu_write_cfg_2",
          "name": "write_file",
          "input": {
            "path": "internal/config/config.go"
          }
        }
      ]
    },
    {
      "role": "user",
      "blocks": [
        {
          "type": "tool_result",
          "id": "toolu_write_cfg_2",
          "name": "write_file",
          "content": "File internal/config/config.go saved successfully."
        }
      ]
    },
    {
      "role": "assistant",
      "content": "What is the next task?"
    }
  ]
}`;

export const Sandbox: React.FC = () => {
  const [inputPrompt, setInputPrompt] = useState(SAMPLE_BLOATED_PAYLOAD);
  const [model, setModel] = useState('claude-3-5-sonnet');
  const [tier1, setTier1] = useState(true);
  const [tier2, setTier2] = useState(true);
  const [tier3, setTier3] = useState(true);
  const [targetRatio, setTargetRatio] = useState(0.60);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<SandboxResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleRunPruning = async () => {
    setLoading(true);
    setError(null);

    try {
      const resp = await fetch('/api/sandbox', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text: inputPrompt,
          model,
          format: inputPrompt.includes('"messages"') ? 'anthropic' : 'raw',
          tier1,
          tier2,
          tier3,
          target_ratio: targetRatio,
        }),
      });

      if (!resp.ok) {
        throw new Error(`Sandbox execution failed: ${resp.statusText}`);
      }

      const data: SandboxResponse = await resp.json();
      setResult(data);
    } catch (err: any) {
      setError(err.message || 'Failed to prune prompt');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-slate-900/60 p-6 rounded-2xl border border-slate-800 shadow-xl backdrop-blur">
        <div>
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-brand-400" />
            Interactive Pruning Sandbox
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Test prompt compression in real-time with sub-millisecond execution
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setInputPrompt(SAMPLE_BLOATED_PAYLOAD)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg transition-colors"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            Reset Sample
          </button>
          <button
            onClick={handleRunPruning}
            disabled={loading || !inputPrompt.trim()}
            className="flex items-center gap-2 px-5 py-2 text-xs font-semibold bg-brand-500 hover:bg-brand-400 text-slate-950 rounded-xl transition-all shadow-lg shadow-brand-500/20 disabled:opacity-50"
          >
            <Play className="w-4 h-4 fill-current" />
            {loading ? 'Pruning...' : 'Run Prune'}
          </button>
        </div>
      </div>

      {/* Configuration Controls Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 bg-slate-900/40 p-4 rounded-xl border border-slate-800/80 text-xs">
        <div>
          <label className="block text-slate-400 mb-1.5 font-medium">Model Calibration</label>
          <select
            value={model}
            onChange={(e) => setModel(e.target.value)}
            className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-slate-200 font-mono focus:border-brand-500 focus:outline-none"
          >
            <option value="claude-3-5-sonnet">claude-3-5-sonnet ($3/M)</option>
            <option value="claude-3-7-sonnet">claude-3-7-sonnet ($3/M)</option>
            <option value="claude-3-5-haiku">claude-3-5-haiku ($0.80/M)</option>
            <option value="gpt-4o">gpt-4o ($2.50/M)</option>
            <option value="gpt-4o-mini">gpt-4o-mini ($0.15/M)</option>
          </select>
        </div>

        <div>
          <label className="block text-slate-400 mb-1.5 font-medium">
            Target Ratio ({Math.round(targetRatio * 100)}%)
          </label>
          <input
            type="range"
            min="0.10"
            max="0.85"
            step="0.05"
            value={targetRatio}
            onChange={(e) => setTargetRatio(parseFloat(e.target.value))}
            className="w-full accent-brand-500 mt-2"
          />
        </div>

        <div className="flex items-center gap-2 pt-4">
          <input
            type="checkbox"
            id="t1"
            checked={tier1}
            onChange={(e) => setTier1(e.target.checked)}
            className="rounded accent-brand-500 w-4 h-4"
          />
          <label htmlFor="t1" className="text-slate-300 font-medium cursor-pointer">
            Tier 1: Syntactic
          </label>
        </div>

        <div className="flex items-center gap-2 pt-4">
          <input
            type="checkbox"
            id="t2"
            checked={tier2}
            onChange={(e) => setTier2(e.target.checked)}
            className="rounded accent-brand-500 w-4 h-4"
          />
          <label htmlFor="t2" className="text-slate-300 font-medium cursor-pointer">
            Tier 2: Tool Eviction
          </label>
        </div>

        <div className="flex items-center gap-2 pt-4">
          <input
            type="checkbox"
            id="t3"
            checked={tier3}
            onChange={(e) => setTier3(e.target.checked)}
            className="rounded accent-brand-500 w-4 h-4"
          />
          <label htmlFor="t3" className="text-slate-300 font-medium cursor-pointer">
            Tier 3: State Delta
          </label>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-950/40 border border-rose-800 rounded-xl text-rose-300 text-xs font-mono">
          {error}
        </div>
      )}

      {/* Input Editor */}
      <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
        <div className="px-6 py-3 bg-slate-950 border-b border-slate-800 flex items-center justify-between text-xs">
          <span className="font-semibold text-slate-300">Inbound Prompt Payload (Raw Text or JSON)</span>
          <span className="text-slate-500 font-mono">{inputPrompt.length} chars</span>
        </div>
        <textarea
          rows={10}
          value={inputPrompt}
          onChange={(e) => setInputPrompt(e.target.value)}
          placeholder="Paste OpenAI or Anthropic payload here..."
          className="w-full p-4 bg-slate-950/80 text-xs font-mono text-slate-200 focus:outline-none resize-y leading-relaxed"
        />
      </div>

      {/* Results View */}
      {result && (
        <div className="flex flex-col gap-6 mt-2">
          {/* Metrics summary bar */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <div className="p-4 bg-slate-900/60 border border-slate-800 rounded-xl">
              <div className="text-xs text-slate-400">Tokens Before / After</div>
              <div className="text-lg font-bold font-mono text-slate-100 mt-1">
                {result.stats.inbound_tokens} → {result.stats.outbound_tokens}
              </div>
            </div>

            <div className="p-4 bg-slate-900/60 border border-slate-800 rounded-xl">
              <div className="text-xs text-slate-400">Compression Achieved</div>
              <div className="text-lg font-bold font-mono text-emerald-400 mt-1">
                {result.stats.compression_ratio.toFixed(1)}% ({result.stats.tokens_saved} saved)
              </div>
            </div>

            <div className="p-4 bg-slate-900/60 border border-slate-800 rounded-xl">
              <div className="text-xs text-slate-400">Processing Overhead</div>
              <div className="text-lg font-bold font-mono text-brand-300 mt-1">
                {result.stats.processing_latency_ms.toFixed(2)} ms
              </div>
            </div>

            <div className="p-4 bg-slate-900/60 border border-slate-800 rounded-xl">
              <div className="text-xs text-slate-400">Cost Savings / Call</div>
              <div className="text-lg font-bold font-mono text-emerald-300 mt-1">
                +${result.usd_saved.toFixed(5)}
              </div>
            </div>
          </div>

          {/* Detailed Diff Viewer */}
          <DiffViewer
            original={result.original_text}
            pruned={result.pruned_text}
            diffs={result.stats.diffs}
            tier1Saved={result.stats.tier1_tokens_saved}
            tier2Saved={result.stats.tier2_tokens_saved}
            tier3Saved={result.stats.tier3_tokens_saved}
          />
        </div>
      )}
    </div>
  );
};
