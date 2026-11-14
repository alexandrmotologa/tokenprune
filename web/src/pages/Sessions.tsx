import React, { useState } from 'react';
import { SessionRecord } from '../types';
import { DiffViewer } from '../components/DiffViewer';
import { Search, ChevronRight, X, Copy, Check, Download, Terminal } from 'lucide-react';

interface SessionsProps {
  sessions: SessionRecord[];
  selectedSession: SessionRecord | null;
  onSelectSession: (s: SessionRecord | null) => void;
}

export const Sessions: React.FC<SessionsProps> = ({
  sessions,
  selectedSession,
  onSelectSession,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const filtered = sessions.filter(s =>
    s.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
    s.session_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
    s.model.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const copyAsCurl = (s: SessionRecord, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    const endpoint = s.provider === 'openai' ? 'http://localhost:8080/v1/chat/completions' : 'http://localhost:8080/v1/messages';
    const payload = s.pruned_preview || s.raw_preview || '{}';
    const cmd = `curl -X POST ${endpoint} \\\n  -H "Content-Type: application/json" \\\n  -d '${payload.replace(/'/g, "'\\''")}'`;

    navigator.clipboard.writeText(cmd);
    setCopiedId(s.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const exportData = (format: 'json' | 'csv') => {
    let content = '';
    let filename = `tokenprune_sessions_${new Date().toISOString().slice(0, 10)}`;
    let type = '';

    if (format === 'json') {
      content = JSON.stringify(filtered, null, 2);
      filename += '.json';
      type = 'application/json';
    } else {
      const headers = ['id', 'session_id', 'created_at', 'provider', 'model', 'inbound_tokens', 'outbound_tokens', 'tokens_saved', 'compression_ratio', 'latency_ms', 'usd_saved'];
      const rows = filtered.map(s => [
        s.id, s.session_id, s.created_at, s.provider, s.model,
        s.inbound_tokens, s.outbound_tokens, s.tokens_saved,
        s.compression_ratio.toFixed(2), s.processing_latency_ms.toFixed(2), s.usd_saved.toFixed(5)
      ].join(','));
      content = [headers.join(','), ...rows].join('\n');
      filename += '.csv';
      type = 'text/csv';
    }

    const blob = new Blob([content], { type });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Header and Controls */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-slate-900/60 p-6 rounded-2xl border border-slate-800 shadow-xl backdrop-blur">
        <div>
          <h2 className="text-lg font-bold text-white">Agent Sessions Archive</h2>
          <p className="text-xs text-slate-400 mt-1">
            Complete audit trail of requests, compression ratios, and latency
          </p>
        </div>

        <div className="flex items-center gap-3 w-full sm:w-auto">
          <div className="relative flex-1 sm:w-64">
            <Search className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Filter sessions..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-brand-500"
            />
          </div>

          <button
            onClick={() => exportData('json')}
            className="flex items-center gap-1.5 px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs transition-colors"
            title="Export as JSON"
          >
            <Download className="w-3.5 h-3.5" />
            JSON
          </button>
          <button
            onClick={() => exportData('csv')}
            className="flex items-center gap-1.5 px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs transition-colors"
            title="Export as CSV"
          >
            <Download className="w-3.5 h-3.5" />
            CSV
          </button>
        </div>
      </div>

      {/* Main Table */}
      <div className="bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-950/80 text-slate-400 font-semibold border-b border-slate-800">
              <tr>
                <th className="px-6 py-3.5">Timestamp</th>
                <th className="px-6 py-3.5">Session / Model</th>
                <th className="px-6 py-3.5 text-right">Inbound</th>
                <th className="px-6 py-3.5 text-right">Pruned</th>
                <th className="px-6 py-3.5 text-right">Saved %</th>
                <th className="px-6 py-3.5 text-right">Cost Saved</th>
                <th className="px-6 py-3.5 text-right">Latency</th>
                <th className="px-6 py-3.5 text-center">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-mono">
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-6 py-8 text-center text-slate-500 font-sans">
                    No sessions match the current filter.
                  </td>
                </tr>
              ) : (
                filtered.map((s) => (
                  <tr
                    key={s.id}
                    onClick={() => onSelectSession(s)}
                    className="hover:bg-slate-800/40 cursor-pointer transition-colors"
                  >
                    <td className="px-6 py-4 text-slate-400 whitespace-nowrap">
                      {new Date(s.created_at).toLocaleTimeString()}
                    </td>
                    <td className="px-6 py-4">
                      <div className="font-semibold text-slate-200">{s.model}</div>
                      <div className="text-[10px] text-slate-500 truncate max-w-[140px]">
                        {s.session_id}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-right text-slate-400">
                      {s.inbound_tokens.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-right text-slate-200">
                      {s.outbound_tokens.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <span className="font-bold text-emerald-400">
                        {s.compression_ratio.toFixed(1)}%
                      </span>
                    </td>
                    <td className="px-6 py-4 text-right text-emerald-300">
                      +${s.usd_saved.toFixed(4)}
                    </td>
                    <td className="px-6 py-4 text-right text-slate-400">
                      {s.processing_latency_ms.toFixed(1)}ms
                    </td>
                    <td className="px-6 py-4 text-center">
                      <div className="flex items-center justify-center gap-1.5">
                        <button
                          onClick={(e) => copyAsCurl(s, e)}
                          title="Copy as cURL command"
                          className="p-1.5 hover:bg-slate-700/60 text-slate-400 hover:text-white rounded-lg transition-colors"
                        >
                          {copiedId === s.id ? (
                            <Check className="w-3.5 h-3.5 text-emerald-400" />
                          ) : (
                            <Terminal className="w-3.5 h-3.5" />
                          )}
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onSelectSession(s);
                          }}
                          className="p-1.5 hover:bg-slate-700/60 text-slate-400 hover:text-white rounded-lg transition-colors"
                        >
                          <ChevronRight className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Detail Modal */}
      {selectedSession && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-fade-in">
          <div className="relative w-full max-w-4xl max-h-[90vh] overflow-y-auto bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl p-6 flex flex-col gap-6">
            <div className="flex items-center justify-between border-b border-slate-800 pb-4">
              <div>
                <div className="text-xs font-mono text-slate-500 uppercase tracking-wider">
                  Session Inspection
                </div>
                <h3 className="text-lg font-bold text-white font-mono mt-0.5">
                  {selectedSession.id}
                </h3>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={(e) => copyAsCurl(selectedSession, e)}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-sans transition-colors"
                >
                  {copiedId === selectedSession.id ? (
                    <>
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                      Copied cURL!
                    </>
                  ) : (
                    <>
                      <Copy className="w-3.5 h-3.5" />
                      Copy cURL
                    </>
                  )}
                </button>
                <button
                  onClick={() => onSelectSession(null)}
                  className="p-2 hover:bg-slate-800 text-slate-400 hover:text-white rounded-xl transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>

            {/* Quick Metrics */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                <div className="text-[11px] text-slate-400">Compression</div>
                <div className="text-base font-bold font-mono text-emerald-400 mt-0.5">
                  {selectedSession.compression_ratio.toFixed(1)}%
                </div>
              </div>
              <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                <div className="text-[11px] text-slate-400">Tokens Eliminated</div>
                <div className="text-base font-bold font-mono text-slate-200 mt-0.5">
                  {selectedSession.tokens_saved.toLocaleString()} tok
                </div>
              </div>
              <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                <div className="text-[11px] text-slate-400">Pruning Overhead</div>
                <div className="text-base font-bold font-mono text-brand-300 mt-0.5">
                  {selectedSession.processing_latency_ms.toFixed(2)} ms
                </div>
              </div>
              <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                <div className="text-[11px] text-slate-400">Direct Cost Saved</div>
                <div className="text-base font-bold font-mono text-emerald-300 mt-0.5">
                  +${selectedSession.usd_saved.toFixed(5)}
                </div>
              </div>
            </div>

            {/* Payload Previews */}
            <DiffViewer
              original={selectedSession.raw_preview || 'No raw preview stored for this turn.'}
              pruned={selectedSession.pruned_preview || 'No pruned preview stored for this turn.'}
              tier1Saved={selectedSession.tier1_saved}
              tier2Saved={selectedSession.tier2_saved}
              tier3Saved={selectedSession.tier3_saved}
            />
          </div>
        </div>
      )}
    </div>
  );
};
