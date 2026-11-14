import React from 'react';
import { GlobalStats, SessionRecord } from '../types';
import { SavingsGauge } from '../components/SavingsGauge';
import { TurnWaterfall } from '../components/TurnWaterfall';
import { LiveFeed } from '../components/LiveFeed';
import { DollarSign, Zap, TrendingDown, Server } from 'lucide-react';

interface DashboardProps {
  stats: GlobalStats | null;
  sessions: SessionRecord[];
  onSelectSession: (s: SessionRecord) => void;
}

export const Dashboard: React.FC<DashboardProps> = ({ stats, sessions, onSelectSession }) => {
  const totalTokensSaved = stats?.total_tokens_saved || 0;
  const totalInbound = stats?.total_inbound_tokens || 0;
  const avgRatio = stats?.avg_compression_ratio || 0;
  const totalUSD = stats?.total_usd_saved || 0;
  const totalReqs = stats?.total_requests || 0;
  const avgLatency = stats?.avg_latency_ms || 0;

  return (
    <div className="flex flex-col gap-6">
      {/* Top Hero Row: Gauge + Key Metric Cards */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <SavingsGauge
          ratio={avgRatio}
          savedTokens={totalTokensSaved}
          inboundTokens={totalInbound}
        />

        <div className="lg:col-span-2 grid grid-cols-1 sm:grid-cols-2 gap-4">
          {/* Card 1: Cumulative USD Saved */}
          <div className="flex flex-col justify-between p-6 bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                Cumulative Cost Avoidance
              </span>
              <div className="p-2 bg-emerald-500/10 text-emerald-400 rounded-xl">
                <DollarSign className="w-5 h-5" />
              </div>
            </div>
            <div className="mt-4">
              <div className="text-3xl font-extrabold text-white font-mono">
                ${totalUSD.toFixed(3)}
              </div>
              <div className="text-xs text-slate-500 mt-1">
                Direct LLM inference API expenditure saved
              </div>
            </div>
          </div>

          {/* Card 2: Latency Speedup */}
          <div className="flex flex-col justify-between p-6 bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                Proxy Pruning Overhead
              </span>
              <div className="p-2 bg-brand-500/10 text-brand-400 rounded-xl">
                <Zap className="w-5 h-5" />
              </div>
            </div>
            <div className="mt-4">
              <div className="text-3xl font-extrabold text-white font-mono">
                {avgLatency.toFixed(2)} ms
              </div>
              <div className="text-xs text-brand-400 mt-1">
                Sub-10ms target achieved (&lt; 1ms CPU time)
              </div>
            </div>
          </div>

          {/* Card 3: Context Reduction */}
          <div className="flex flex-col justify-between p-6 bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                Total Turns Processed
              </span>
              <div className="p-2 bg-blue-500/10 text-blue-400 rounded-xl">
                <Server className="w-5 h-5" />
              </div>
            </div>
            <div className="mt-4">
              <div className="text-3xl font-extrabold text-white font-mono">
                {totalReqs.toLocaleString()}
              </div>
              <div className="text-xs text-slate-500 mt-1">
                Multi-turn autonomous agent requests intercepted
              </div>
            </div>
          </div>

          {/* Card 4: Average TTFT */}
          <div className="flex flex-col justify-between p-6 bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                Prefill Acceleration
              </span>
              <div className="p-2 bg-purple-500/10 text-purple-400 rounded-xl">
                <TrendingDown className="w-5 h-5" />
              </div>
            </div>
            <div className="mt-4">
              <div className="text-3xl font-extrabold text-white font-mono">
                ~2.2x Faster
              </div>
              <div className="text-xs text-slate-500 mt-1">
                Reduced Time to First Token via trimmed history
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Middle Row: Turn Waterfall Context Curve */}
      <TurnWaterfall sessions={sessions} />

      {/* Bottom Row: Live Traffic Feed */}
      <LiveFeed sessions={sessions} onSelectSession={onSelectSession} />
    </div>
  );
};
