import React from 'react';
import { SessionRecord } from '../types';
import { Clock } from 'lucide-react';

interface LiveFeedProps {
  sessions: SessionRecord[];
  onSelectSession: (session: SessionRecord) => void;
}

export const LiveFeed: React.FC<LiveFeedProps> = ({ sessions, onSelectSession }) => {
  return (
    <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur overflow-hidden">
      <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
        <div className="flex items-center gap-2.5">
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-brand-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-brand-500"></span>
          </span>
          <h3 className="text-sm font-semibold text-slate-200">Live Traffic Feed</h3>
        </div>
        <span className="text-xs text-slate-500 font-mono">
          {sessions.length} turns logged
        </span>
      </div>

      <div className="divide-y divide-slate-800/60 max-h-[420px] overflow-y-auto">
        {sessions.length === 0 ? (
          <div className="p-8 text-center text-slate-500 text-sm">
            Waiting for proxy requests on :8080...
          </div>
        ) : (
          sessions.map((s) => {
            const timeStr = new Date(s.created_at).toLocaleTimeString();
            return (
              <div
                key={s.id}
                onClick={() => onSelectSession(s)}
                className="flex items-center justify-between p-4 hover:bg-slate-800/40 cursor-pointer transition-colors group"
              >
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-slate-200 font-mono group-hover:text-brand-400 transition-colors">
                      {s.model}
                    </span>
                    <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono uppercase">
                      {s.provider}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-slate-500">
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3 text-slate-600" />
                      {timeStr}
                    </span>
                    <span>•</span>
                    <span>{s.processing_latency_ms.toFixed(1)}ms prune</span>
                    {s.upstream_ttft_ms > 0 && (
                      <>
                        <span>•</span>
                        <span>{s.upstream_ttft_ms.toFixed(0)}ms TTFT</span>
                      </>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-6 text-right">
                  <div className="flex flex-col items-end">
                    <span className="text-sm font-bold font-mono text-emerald-400">
                      -{s.tokens_saved.toLocaleString()} tok
                    </span>
                    <span className="text-xs text-slate-400 font-medium">
                      {s.compression_ratio.toFixed(1)}% savings
                    </span>
                  </div>

                  <div className="hidden sm:flex flex-col items-end">
                    <span className="text-xs font-medium font-mono text-emerald-300">
                      +${s.usd_saved.toFixed(4)}
                    </span>
                    <span className="text-[11px] text-slate-500">
                      {(s.outbound_tokens / 1000).toFixed(1)}k sent
                    </span>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};
