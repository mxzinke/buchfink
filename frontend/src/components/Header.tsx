import React, { useState } from 'react';
import { Calendar, Plus } from 'lucide-react';

interface HeaderProps {
  currentYear: number;
  availableYears: number[];
  onYearChange: (year: number) => void;
  onCreateYear: (year: number) => void;
}

export const Header: React.FC<HeaderProps> = ({
  currentYear,
  availableYears,
  onYearChange,
  onCreateYear,
}) => {
  const [isAddingYear, setIsAddingYear] = useState(false);
  const [newYearInput, setNewYearInput] = useState(currentYear + 1);

  const handleAddYearSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newYearInput >= 2000 && newYearInput <= 2100) {
      onCreateYear(newYearInput);
      setIsAddingYear(false);
    }
  };

  return (
    <header className="h-14 border-b border-stone-200/90 bg-[#FAF9F6]/90 backdrop-blur-md px-6 flex items-center justify-between select-none sticky top-0 z-30 window-drag">
      {/* Left / Fiscal Year Selector */}
      <div className="flex items-center gap-3 window-no-drag">
        <span className="text-xs font-semibold text-stone-500 uppercase tracking-wider">
          Geschäftsjahr:
        </span>
        <div className="flex items-center gap-1 bg-stone-200/60 p-1 rounded-lg border border-stone-300/60 text-xs">
          <Calendar className="w-3.5 h-3.5 text-stone-500 ml-1 mr-0.5" />
          {availableYears.map((year) => (
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

          {isAddingYear ? (
            <form onSubmit={handleAddYearSubmit} className="flex items-center gap-1 ml-1">
              <input
                type="number"
                min="2000"
                max="2100"
                value={newYearInput}
                onChange={(e) => setNewYearInput(Number(e.target.value))}
                className="w-16 px-1.5 py-0.5 bg-white border border-stone-300 rounded text-xs font-mono font-bold"
                autoFocus
              />
              <button
                type="submit"
                className="px-2 py-0.5 bg-amber-600 text-white rounded font-bold text-xs"
              >
                +
              </button>
              <button
                type="button"
                onClick={() => setIsAddingYear(false)}
                className="px-1.5 py-0.5 text-stone-500 hover:text-stone-800 text-xs"
              >
                ✕
              </button>
            </form>
          ) : (
            <button
              onClick={() => setIsAddingYear(true)}
              className="p-1 text-stone-500 hover:text-stone-900 hover:bg-stone-300/50 rounded-md transition-colors ml-0.5"
              title="Weiteres Geschäftsjahr anlegen"
            >
              <Plus className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>
    </header>
  );
};
