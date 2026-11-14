import React from 'react';

interface SavingsGaugeProps {
  ratio: number; // 0 to 100
  savedTokens: number;
  inboundTokens: number;
}

export const SavingsGauge: React.FC<SavingsGaugeProps> = ({
  ratio,
  savedTokens,
  inboundTokens,
}) => {
  const clamped = Math.min(Math.max(ratio, 0), 100);
  const radius = 80;
  const circumference = 2 * Math.PI * radius;
  // Semi-circle gauge (180 degrees)
  const arcLength = circumference * 0.75;
  const strokeDashoffset = arcLength - (arcLength * clamped) / 100;

  const formatNumber = (num: number) => {
    if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
    if (num >= 1_000) return `${(num / 1_000).toFixed(1)}k`;
    return num.toString();
  };

  return (
    <div className="relative flex flex-col items-center justify-center p-6 bg-slate-900/60 border border-slate-800 rounded-2xl shadow-xl backdrop-blur">
      <div className="text-xs font-semibold tracking-wider uppercase text-slate-400 mb-2">
        Semantic Compression Rate
      </div>

      <div className="relative w-52 h-44 flex items-center justify-center">
        <svg className="w-52 h-52 -rotate-135 transform" viewBox="0 0 200 200">
          {/* Background Track */}
          <circle
            cx="100"
            cy="100"
            r={radius}
            className="text-slate-800"
            strokeWidth="14"
            strokeDasharray={`${arcLength} ${circumference}`}
            strokeLinecap="round"
            stroke="currentColor"
            fill="transparent"
          />
          {/* Active Gradient Meter */}
          <circle
            cx="100"
            cy="100"
            r={radius}
            className="text-brand-500 transition-all duration-1000 ease-out"
            strokeWidth="14"
            strokeDasharray={`${arcLength} ${circumference}`}
            strokeDashoffset={strokeDashoffset}
            strokeLinecap="round"
            stroke="currentColor"
            fill="transparent"
          />
        </svg>

        {/* Center Readout */}
        <div className="absolute flex flex-col items-center justify-center text-center mt-2">
          <span className="text-4xl font-bold tracking-tight text-white font-mono">
            {clamped.toFixed(1)}%
          </span>
          <span className="text-xs font-medium text-brand-400 mt-0.5">
            tokens eliminated
          </span>
        </div>
      </div>

      <div className="w-full grid grid-cols-2 gap-4 mt-2 pt-4 border-t border-slate-800/80 text-center">
        <div>
          <div className="text-xs text-slate-500">Pruned Volume</div>
          <div className="text-sm font-semibold font-mono text-slate-200 mt-0.5">
            {formatNumber(savedTokens)} tokens
          </div>
        </div>
        <div>
          <div className="text-xs text-slate-500">Inbound Flow</div>
          <div className="text-sm font-semibold font-mono text-slate-200 mt-0.5">
            {formatNumber(inboundTokens)} tokens
          </div>
        </div>
      </div>
    </div>
  );
};
