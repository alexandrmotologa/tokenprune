import React, { useEffect, useState } from 'react';
import { GlobalStats, SessionRecord } from './types';
import { Dashboard } from './pages/Dashboard';
import { Sessions } from './pages/Sessions';
import { Sandbox } from './components/Sandbox';
import { Settings } from './pages/Settings';
import {
  Scissors,
  LayoutDashboard,
  ListOrdered,
  Sparkles,
  Settings as SettingsIcon,
  Github,
} from 'lucide-react';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'sessions' | 'sandbox' | 'settings'>('dashboard');
  const [stats, setStats] = useState<GlobalStats | null>(null);
  const [sessions, setSessions] = useState<SessionRecord[]>([]);
  const [selectedSession, setSelectedSession] = useState<SessionRecord | null>(null);
  const [connected, setConnected] = useState(false);

  // Initial data fetch
  const refreshData = async () => {
    try {
      const [statsRes, sessRes] = await Promise.all([
        fetch('/api/stats'),
        fetch('/api/sessions?limit=50'),
      ]);
      if (statsRes.ok) {
        const s = await statsRes.json();
        setStats(s);
      }
      if (sessRes.ok) {
        const sess = await sessRes.json();
        setSessions(sess);
      }
    } catch (err) {
      console.error('Failed to fetch initial telemetry data', err);
    }
  };

  useEffect(() => {
    refreshData();

    // SSE EventSource connection for live stream updates
    const eventSource = new EventSource('/api/events');

    eventSource.onopen = () => {
      setConnected(true);
    };

    eventSource.addEventListener('session', (e) => {
      try {
        const newRecord: SessionRecord = JSON.parse(e.data);
        setSessions((prev) => [newRecord, ...prev.slice(0, 99)]);
        // Incrementally update global stats
        setStats((prev) => {
          if (!prev) return null;
          const totalReqs = prev.total_requests + 1;
          const totalInbound = prev.total_inbound_tokens + newRecord.inbound_tokens;
          const totalOutbound = prev.total_outbound_tokens + newRecord.outbound_tokens;
          const totalSaved = prev.total_tokens_saved + newRecord.tokens_saved;
          const totalUSD = prev.total_usd_saved + newRecord.usd_saved;
          return {
            ...prev,
            total_requests: totalReqs,
            total_inbound_tokens: totalInbound,
            total_outbound_tokens: totalOutbound,
            total_tokens_saved: totalSaved,
            total_usd_saved: totalUSD,
            avg_compression_ratio: totalInbound > 0 ? (totalSaved / totalInbound) * 100 : 0,
          };
        });
      } catch (err) {
        console.error('Failed to parse SSE event', err);
      }
    });

    eventSource.onerror = () => {
      setConnected(false);
    };

    return () => {
      eventSource.close();
    };
  }, []);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col">
      {/* Top Navigation Bar */}
      <header className="sticky top-0 z-40 bg-slate-950/80 border-b border-slate-800/80 backdrop-blur-md">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-9 h-9 rounded-xl bg-gradient-to-tr from-brand-600 to-emerald-400 text-slate-950 shadow-lg shadow-brand-500/20">
              <Scissors className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold tracking-tight text-base font-mono text-white">
                  tokenprune
                </span>
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-brand-500/10 text-brand-400 font-mono font-semibold border border-brand-500/20">
                  v1.0.0
                </span>
              </div>
              <div className="text-[10px] text-slate-500 hidden sm:block">
                Streaming Semantic Context Compressor
              </div>
            </div>
          </div>

          {/* Navigation Tabs */}
          <nav className="flex items-center gap-1 bg-slate-900/80 p-1 rounded-xl border border-slate-800/80 text-xs">
            <button
              onClick={() => setActiveTab('dashboard')}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
                activeTab === 'dashboard'
                  ? 'bg-slate-800 text-white font-medium shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <LayoutDashboard className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Dashboard</span>
            </button>
            <button
              onClick={() => setActiveTab('sessions')}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
                activeTab === 'sessions'
                  ? 'bg-slate-800 text-white font-medium shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <ListOrdered className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Sessions</span>
            </button>
            <button
              onClick={() => setActiveTab('sandbox')}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
                activeTab === 'sandbox'
                  ? 'bg-slate-800 text-white font-medium shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Sparkles className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Sandbox</span>
            </button>
            <button
              onClick={() => setActiveTab('settings')}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
                activeTab === 'settings'
                  ? 'bg-slate-800 text-white font-medium shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <SettingsIcon className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Settings</span>
            </button>
          </nav>

          {/* Connection status & GitHub */}
          <div className="flex items-center gap-3">
            <div className="hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-full bg-slate-900 border border-slate-800 text-xs">
              <span
                className={`w-2 h-2 rounded-full ${
                  connected ? 'bg-brand-500 animate-pulse' : 'bg-rose-500'
                }`}
              ></span>
              <span className="text-[11px] text-slate-400 font-mono">
                {connected ? 'Live SSE' : 'Offline'}
              </span>
            </div>

            <a
              href="https://github.com/alexandrmotologa/tokenprune"
              target="_blank"
              rel="noreferrer"
              className="p-2 text-slate-400 hover:text-white rounded-xl hover:bg-slate-900 transition-colors"
            >
              <Github className="w-5 h-5" />
            </a>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'dashboard' && (
          <Dashboard
            stats={stats}
            sessions={sessions}
            onSelectSession={(s) => {
              setSelectedSession(s);
              setActiveTab('sessions');
            }}
          />
        )}
        {activeTab === 'sessions' && (
          <Sessions
            sessions={sessions}
            selectedSession={selectedSession}
            onSelectSession={setSelectedSession}
          />
        )}
        {activeTab === 'sandbox' && <Sandbox />}
        {activeTab === 'settings' && <Settings />}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-900 py-6 text-center text-xs text-slate-600 font-mono">
        tokenprune • High-Speed Semantic Context Compressor & Streaming Reverse Proxy
      </footer>
    </div>
  );
};
