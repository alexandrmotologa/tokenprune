import React, { useState, useEffect } from 'react';
import { SandboxResponse } from '../types';
import { DiffViewer } from './DiffViewer';
import { Play, Sparkles, RotateCcw, Bookmark } from 'lucide-react';

const SCENARIO_MULTI_FILE = `{
  "model": "claude-3-5-sonnet",
  "system": "CRITICAL_RULES: Maintain zero-loss accuracy on active working context.",
  "messages": [
    {
      "role": "user",
      "content": "Please read config.go and refactor the pool limits."
    },
    {
      "role": "assistant",
      "content": [
        {"type": "text", "text": "Reading config/database.go."},
        {
          "type": "tool_use",
          "id": "toolu_read_1",
          "name": "read_file",
          "input": {"path": "config/database.go"}
        }
      ]
    },
    {
      "role": "user",
      "content": [
        {
          "type": "tool_result",
          "tool_use_id": "toolu_read_1",
          "name": "read_file",
          "content": "package config\\n\\n// Verbose file lines that bloat conversation context.\\n// Line 1: Pool setting\\n// Line 2: Pool setting\\n// Line 3: Pool setting\\n// Line 4: Pool setting\\n// Line 5: Pool setting\\n// Line 6: Pool setting\\n// Line 7: Pool setting\\n// Line 8: Pool setting\\n// Line 9: Pool setting\\n// Line 10: Pool setting\\n"
        }
      ]
    },
    {
      "role": "assistant",
      "content": [
        {"type": "text", "text": "Writing updated database pool configuration."},
        {
          "type": "tool_use",
          "id": "toolu_write_1",
          "name": "write_file",
          "input": {"path": "config/database.go"}
        }
      ]
    },
    {
      "role": "user",
      "content": [
        {
          "type": "tool_result",
          "tool_use_id": "toolu_write_1",
          "name": "write_file",
          "content": "File config/database.go updated successfully."
        }
      ]
    },
    {
      "role": "assistant",
      "content": "Updated pool limits. What is next?"
    }
  ]
}`;

const SCENARIO_ERROR_PRESERVE = `{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a backend test runner. CRITICAL_RULES: Report test failures."},
    {"role": "user", "content": "Run backend unit tests."},
    {
      "role": "assistant",
      "tool_calls": [
        {
          "id": "call_test_1",
          "type": "function",
          "function": {"name": "run_test", "arguments": "{\\"package\\": \\"./pkg/auth\\"}"}
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_test_1",
      "name": "run_test",
      "content": "--- FAIL: TestAuthValidation (0.02s)\\n    auth_test.go:54: expected 200 OK, got 401 Unauthorized\\nFAIL\\nexit status 1"
    },
    {"role": "assistant", "content": "The auth test failed. Fixing token authorization header."},
    {"role": "user", "content": "Please verify the error and retry."}
  ]
}`;

const SCENARIO_TIER1_MARKUP = `{
  "model": "claude-3-5-sonnet",
  "messages": [
    {
      "role": "user",
      "content": "Summarize user profile schema."
    },
    {
      "role": "assistant",
      "content": "<!-- internal telemetry comment: pipeline_v1_run -->\\nHere is the schema:\\n\\n{\\n  \\"user_id\\": 42,\\n  \\"permissions\\": [\\n    \\"read\\",\\n    \\"write\\",\\n    \\"admin\\"\\n  ],\\n  \\"metadata\\": {\\n    \\"tier\\": \\"enterprise\\"\\n  }\\n}\\n\\n|   Column A   |   Column B   |\\n|--------------|--------------|\\n|    value 1   |    value 2   |\\n\\n\`\`\`python\\ndef execute():\\n    # Indentation inside code blocks is strictly preserved\\n    return True\\n\`\`\`"
    }
  ]
}`;

export const Sandbox: React.FC = () => {
  const [inputPrompt, setInputPrompt] = useState(SCENARIO_MULTI_FILE);
  const [model, setModel] = useState('claude-3-5-sonnet');
  const [tier1, setTier1] = useState(true);
  const [tier2, setTier2] = useState(true);
  const [tier3, setTier3] = useState(true);
  const [targetRatio, setTargetRatio] = useState(0.60);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<SandboxResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    handleRunPruning();
  }, []);

  const loadScenario = (preset: string) => {
    switch (preset) {
      case 'multi-file':
        setInputPrompt(SCENARIO_MULTI_FILE);
        setModel('claude-3-5-sonnet');
        setTier1(true);
        setTier2(true);
        setTier3(true);
        break;
      case 'error-preserve':
        setInputPrompt(SCENARIO_ERROR_PRESERVE);
        setModel('gpt-4o');
        setTier1(true);
        setTier2(true);
        setTier3(true);
        break;
      case 'markup':
        setInputPrompt(SCENARIO_TIER1_MARKUP);
        setModel('claude-3-5-sonnet');
        setTier1(true);
        setTier2(false);
        setTier3(false);
        break;
    }
    setResult(null);
  };

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
          format: inputPrompt.includes('"messages"') ? (inputPrompt.includes('"tool_calls"') ? 'openai' : 'anthropic') : 'raw',
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

        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1.5 bg-slate-950 px-2.5 py-1.5 rounded-lg border border-slate-800 text-xs">
            <Bookmark className="w-3.5 h-3.5 text-slate-500" />
            <select
              onChange={(e) => loadScenario(e.target.value)}
              className="bg-transparent text-slate-300 focus:outline-none cursor-pointer"
              defaultValue="multi-file"
            >
              <option value="multi-file" className="bg-slate-950">Preset: Superseded Tool Reads</option>
              <option value="error-preserve" className="bg-slate-950">Preset: Preserved Error Output</option>
              <option value="markup" className="bg-slate-950">Preset: JSON & Table Padding</option>
            </select>
          </div>

          <button
            onClick={() => loadScenario('multi-file')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg transition-colors"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            Reset
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
