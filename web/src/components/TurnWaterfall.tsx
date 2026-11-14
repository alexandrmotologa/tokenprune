import React, { useState } from 'react';
import { SessionRecord } from '../types';

interface TurnWaterfallProps {
  sessions: SessionRecord[];
}

export const TurnWaterfall: React.FC<TurnWaterfallProps> = ({ sessions }) => {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);

  // Take the most recent 12 sessions in chronological order
  const data = sessions.slice(0, 14).reverse();

  if (data.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-slate-900/60 border border-slate-800 rounded-2xl h-80 text-slate-500">
        <span>No proxy traffic recorded yet</span>
        <span className="text-xs text-slate-600 mt-1">Send requests through :8080 to see live context curves</span>
      </div>
    );
  }

  const maxToken = Math.max(...data.map(d => Math.max(d.inbound_tokens, d.outbound_tokens)), 1000);

  const width = 640;
  const height = 220;
  const paddingX = 40;
  const paddingY = 25;

  const chartW = width - paddingX * 2;
  const chartH = height - paddingY * 2;

  const getX = (i: number) => paddingX + (i / Math.max(data.length - 1, 1)) * chartW;
  const getY = (val: number) => height - paddingY - (val / maxToken) * chartH;

  // Build SVG polyline points
  const rawPoints = data.map((d, i) => `${getX(i)},${getY(d.inbound_tokens)}`).join(' ');
  const prunedPoints = data.map((d, i) => `${getX(i)},${getY(d.outbound_tokens)}`).join(' ');

  // SVG Area path for savings
  const areaPath = `
    M ${getX(0)} ${getY(data[0].inbound_tokens)}
    ${data.map((d, i) => `L ${getX(i)} ${getY(d.inbound_tokens)}`).join(' ')}
    ${data.slice().reverse().map((d, i) => `L ${getX(data.length - 1 - i)} ${getY(d.outbound_tokens)}`).join(' ')}
    Z
  `;

  return (
    <div className="flex flex-col p-6 bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur relative">
      <div className="flex items-center justify-between mb-4">
        <div>
          <div className="text-sm font-semibold text-slate-200">Context Inflation vs Pruned Payload</div>
          <div className="text-xs text-slate-500 mt-0.5">Prompt token trajectory across sequential agent turns</div>
        </div>
        <div className="flex items-center gap-4 text-xs font-medium">
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-rose-500"></span>
            <span className="text-slate-400">Raw Prompt</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-brand-500"></span>
            <span className="text-slate-400">tokenprune Context</span>
          </div>
        </div>
      </div>

      <div className="relative w-full overflow-hidden">
        <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-56 overflow-visible">
          {/* Grid lines */}
          {[0, 0.25, 0.5, 0.75, 1].map((pct, i) => {
            const y = height - paddingY - pct * chartH;
            const tokenVal = Math.round(pct * maxToken);
            return (
              <g key={i}>
                <line
                  x1={paddingX}
                  y1={y}
                  x2={width - paddingX}
                  y2={y}
                  stroke="#1e293b"
                  strokeDasharray="4 4"
                />
                <text
                  x={paddingX - 8}
                  y={y + 4}
                  fill="#64748b"
                  fontSize="10"
                  textAnchor="end"
                  className="font-mono"
                >
                  {tokenVal >= 1000 ? `${(tokenVal / 1000).toFixed(0)}k` : tokenVal}
                </text>
              </g>
            );
          })}

          {/* Savings Area */}
          <path d={areaPath} fill="rgba(34, 197, 94, 0.12)" />

          {/* Raw Line */}
          <polyline
            fill="none"
            stroke="#f43f5e"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            points={rawPoints}
          />

          {/* Pruned Line */}
          <polyline
            fill="none"
            stroke="#22c55e"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            points={prunedPoints}
          />

          {/* Interactive Data points */}
          {data.map((d, i) => (
            <g
              key={d.id}
              className="cursor-pointer"
              onMouseEnter={() => setHoveredIdx(i)}
              onMouseLeave={() => setHoveredIdx(null)}
            >
              <circle
                cx={getX(i)}
                cy={getY(d.inbound_tokens)}
                r={hoveredIdx === i ? 6 : 3.5}
                className="fill-rose-500 stroke-slate-950 transition-all duration-150"
                strokeWidth="2"
              />
              <circle
                cx={getX(i)}
                cy={getY(d.outbound_tokens)}
                r={hoveredIdx === i ? 6 : 3.5}
                className="fill-brand-500 stroke-slate-950 transition-all duration-150"
                strokeWidth="2"
              />
            </g>
          ))}
        </svg>

        {/* Hover Tooltip */}
        {hoveredIdx !== null && data[hoveredIdx] && (
          <div
            className="absolute top-2 right-2 bg-slate-950/95 border border-slate-700/80 rounded-xl p-3 shadow-2xl text-xs backdrop-blur pointer-events-none z-10 font-mono"
          >
            <div className="font-semibold text-slate-300 mb-1 border-b border-slate-800 pb-1">
              Turn #{hoveredIdx + 1} ({data[hoveredIdx].model})
            </div>
            <div className="text-rose-400">
              Raw: {data[hoveredIdx].inbound_tokens.toLocaleString()} tokens
            </div>
            <div className="text-brand-400">
              Pruned: {data[hoveredIdx].outbound_tokens.toLocaleString()} tokens
            </div>
            <div className="text-slate-400 mt-1 pt-1 border-t border-slate-800 flex justify-between gap-4">
              <span>Saved:</span>
              <span className="font-bold text-emerald-300">
                {data[hoveredIdx].compression_ratio.toFixed(1)}% ({data[hoveredIdx].tokens_saved.toLocaleString()})
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
