import React, { useState } from 'react';
import { DiffItem } from '../types';
import { Layers, Zap, Scissors } from 'lucide-react';

interface DiffViewerProps {
  original: string;
  pruned: string;
  diffs?: DiffItem[];
  tier1Saved?: number;
  tier2Saved?: number;
  tier3Saved?: number;
}

export const DiffViewer: React.FC<DiffViewerProps> = ({
  original,
  pruned,
  diffs = [],
  tier1Saved = 0,
  tier2Saved = 0,
  tier3Saved = 0,
}) => {
  const [selectedTier, setSelectedTier] = useState<number | 'all'>('all');
  const [viewMode, setViewMode] = useState<'side' | 'changes'>('changes');

  const filteredDiffs = selectedTier === 'all'
    ? diffs
    : diffs.filter(d => d.tier === selectedTier);

  return (
    <div className="flex flex-col gap-4">
      {/* Tier Breakdown Chips */}
      <div className="flex flex-wrap items-center justify-between gap-3 bg-slate-900/40 p-3 rounded-xl border border-slate-800">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setSelectedTier('all')}
            className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              selectedTier === 'all'
                ? 'bg-slate-700 text-white shadow'
                : 'bg-slate-800/60 text-slate-400 hover:text-slate-200'
            }`}
          >
            All Modifications ({diffs.length})
          </button>
          <button
            onClick={() => setSelectedTier(1)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              selectedTier === 1
                ? 'bg-blue-600/30 border border-blue-500/50 text-blue-300'
                : 'bg-slate-800/60 text-slate-400 hover:text-slate-200'
            }`}
          >
            <Zap className="w-3.5 h-3.5 text-blue-400" />
            Tier 1 Syntactic ({tier1Saved} tok)
          </button>
          <button
            onClick={() => setSelectedTier(2)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              selectedTier === 2
                ? 'bg-amber-600/30 border border-amber-500/50 text-amber-300'
                : 'bg-slate-800/60 text-slate-400 hover:text-slate-200'
            }`}
          >
            <Scissors className="w-3.5 h-3.5 text-amber-400" />
            Tier 2 Tool Eviction ({tier2Saved} tok)
          </button>
          <button
            onClick={() => setSelectedTier(3)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              selectedTier === 3
                ? 'bg-purple-600/30 border border-purple-500/50 text-purple-300'
                : 'bg-slate-800/60 text-slate-400 hover:text-slate-200'
            }`}
          >
            <Layers className="w-3.5 h-3.5 text-purple-400" />
            Tier 3 State Delta ({tier3Saved} tok)
          </button>
        </div>

        {/* View Switcher */}
        <div className="flex items-center bg-slate-950 p-1 rounded-lg border border-slate-800 text-xs">
          <button
            onClick={() => setViewMode('changes')}
            className={`px-3 py-1 rounded-md transition-all ${
              viewMode === 'changes'
                ? 'bg-slate-800 text-white font-medium shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Tier Actions
          </button>
          <button
            onClick={() => setViewMode('side')}
            className={`px-3 py-1 rounded-md transition-all ${
              viewMode === 'side'
                ? 'bg-slate-800 text-white font-medium shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Side-by-Side
          </button>
        </div>
      </div>

      {viewMode === 'changes' ? (
        <div className="flex flex-col gap-3">
          {filteredDiffs.length === 0 ? (
            <div className="p-8 text-center text-slate-500 bg-slate-900/30 border border-slate-800/60 rounded-xl text-sm">
              No modifications recorded for this selection.
            </div>
          ) : (
            filteredDiffs.map((d, i) => (
              <div
                key={i}
                className="flex flex-col bg-slate-900/80 border border-slate-800 rounded-xl overflow-hidden shadow-lg"
              >
                <div className="flex items-center justify-between px-4 py-2 bg-slate-950 border-b border-slate-800/80 text-xs font-mono">
                  <div className="flex items-center gap-2">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider ${
                        d.tier === 1
                          ? 'bg-blue-500/20 text-blue-300'
                          : d.tier === 2
                          ? 'bg-amber-500/20 text-amber-300'
                          : 'bg-purple-500/20 text-purple-300'
                      }`}
                    >
                      Tier {d.tier}
                    </span>
                    <span className="text-slate-300 font-sans font-medium">
                      {d.description}
                    </span>
                  </div>
                  <span className="text-emerald-400 font-semibold">
                    -{d.tokens_saved} tokens
                  </span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 text-xs font-mono divide-y md:divide-y-0 md:divide-x divide-slate-800">
                  <div className="p-3 bg-rose-950/10 text-rose-300/90 overflow-x-auto max-h-48 whitespace-pre-wrap">
                    <div className="text-[10px] uppercase font-sans font-semibold text-rose-500/80 mb-1">
                      Original Payload
                    </div>
                    {d.original}
                  </div>
                  <div className="p-3 bg-emerald-950/10 text-emerald-300/90 overflow-x-auto max-h-48 whitespace-pre-wrap">
                    <div className="text-[10px] uppercase font-sans font-semibold text-emerald-500/80 mb-1">
                      Pruned Result
                    </div>
                    {d.pruned}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
            <div className="px-4 py-2.5 bg-slate-950 border-b border-slate-800 flex items-center justify-between text-xs">
              <span className="font-semibold text-rose-400 font-mono">Raw Inbound Prompt</span>
              <span className="text-slate-500">{original.length.toLocaleString()} characters</span>
            </div>
            <pre className="p-4 text-xs font-mono text-slate-300 overflow-auto max-h-[500px] whitespace-pre-wrap leading-relaxed">
              {original}
            </pre>
          </div>

          <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
            <div className="px-4 py-2.5 bg-slate-950 border-b border-slate-800 flex items-center justify-between text-xs">
              <span className="font-semibold text-brand-400 font-mono">tokenprune Context</span>
              <span className="text-slate-500">{pruned.length.toLocaleString()} characters</span>
            </div>
            <pre className="p-4 text-xs font-mono text-slate-300 overflow-auto max-h-[500px] whitespace-pre-wrap leading-relaxed">
              {pruned}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
};
