import React, { useState } from 'react';
import { HelpCircle } from 'lucide-react';

interface HelpTooltipProps {
  title: string;
  content: string;
  skrTip?: string;
}

export const HelpTooltip: React.FC<HelpTooltipProps> = ({ title, content, skrTip }) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative inline-flex items-center ml-1.5 align-middle">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        onMouseEnter={() => setIsOpen(true)}
        onMouseLeave={() => setIsOpen(false)}
        className="text-stone-400 hover:text-amber-700 transition-colors p-0.5 rounded-full hover:bg-stone-200/50"
        aria-label={title}
      >
        <HelpCircle className="w-3.5 h-3.5" />
      </button>

      {isOpen && (
        <div className="absolute left-6 top-1/2 -translate-y-1/2 z-50 w-72 p-3 bg-stone-900 text-stone-100 text-xs rounded-lg shadow-xl border border-stone-700/60 pointer-events-none animate-in fade-in zoom-in-95 duration-150">
          <p className="font-semibold text-amber-300 mb-1">{title}</p>
          <p className="text-stone-300 leading-relaxed">{content}</p>
          {skrTip && (
            <div className="mt-2 pt-2 border-t border-stone-800 text-stone-400">
              <span className="font-medium text-amber-400/90">SKR04-Tipp:</span> {skrTip}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
