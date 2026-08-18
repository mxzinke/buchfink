import React from 'react';
import { ShieldCheck, ShieldAlert, Calendar, RefreshCw } from 'lucide-react';
import { IntegrityCheckResult } from '../types';

interface HeaderProps {
  currentYear: number;
  onYearChange: (year: number) => void;
  integrity: IntegrityCheckResult | null;
  onRefreshIntegrity: () => void;
  isCheckingIntegrity: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  currentYear,
  onYearChange,
  integrity,
  onRefreshIntegrity,
  isCheckingIntegrity,
}) => {
  return (
    <header className="h-14 border-b border-stone-200/90 bg-[#FAF9F6]/90 backdrop-blur-md px-6 flex items-center justify-between select-none sticky top-0 z-30 window-drag">
      {/* Left / Fiscal Year Selector */}
      <div className="flex items-center gap-3 window-no-drag">
        <span className="text-xs font-semibold text-stone-500 uppercase tracking-wider">
          Geschäftsjahr:
        </span>
        <div className="flex items-center gap-1 bg-stone-200/60 p-1 rounded-lg border border-stone-300/60 text-xs">
          <Calendar className="w-3.5 h-3.5 text-stone-500 ml-1 mr-0.5" />
          {[2023, 2024, 2025].map((year) => (
            <button
              key={year}
              onClick={() => onYearChange(year)}
              className={`px-2.5 py-0.5 rounded-md font-medium transition-all ${
                currentYear === year
                  ? 'bg-amber-600 text-white shadow-xs font-semibold'
                  : 'text-stone-600 hover:text-stone-900 hover:bg-stone-300/50'
              }`}
            >
              {year}
            </button>
          ))}
        </div>
      </div>

      {/* Right / GoBD Hash Chain Status Badge */}
      <div className="flex items-center gap-2 window-no-drag">
        <div
          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${
            integrity?.isValid
              ? 'bg-emerald-50 text-emerald-800 border-emerald-200'
              : 'bg-rose-50 text-rose-800 border-rose-200'
          }`}
          title={integrity?.message || 'GoBD Hash-Chain Integrität'}
        >
          {integrity?.isValid ? (
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
          ) : (
            <ShieldAlert className="w-3.5 h-3.5 text-rose-600" />
          )}
          <span>{integrity?.isValid ? 'GoBD-Kette intakt' : 'Integritätswarnung'}</span>
        </div>

        <button
          onClick={onRefreshIntegrity}
          disabled={isCheckingIntegrity}
          className="p-1 text-stone-400 hover:text-stone-700 hover:bg-stone-200/70 rounded-lg transition-colors"
          title="Hash-Chain neu validieren"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isCheckingIntegrity ? 'animate-spin text-amber-600' : ''}`} />
        </button>
      </div>
    </header>
  );
};
