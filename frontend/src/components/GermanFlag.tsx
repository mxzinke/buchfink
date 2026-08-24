import React from 'react';

interface GermanFlagProps {
  className?: string;
}

/**
 * Die einzige Stelle mit festen Farbwerten. Schwarz, Rot und Gold sind
 * vorgeschrieben (Bundesflaggenverordnung), also keine Tokens: eine Flagge in
 * Papierfarben wäre keine Flagge mehr. Die Prüfung in `task check:design`
 * nimmt diese Datei deshalb aus.
 */
export const GermanFlag: React.FC<GermanFlagProps> = ({ className = 'w-4 h-3' }) => (
  <span title="Entwickelt für deutsche Standards" className="inline-flex items-center">
    <svg
      viewBox="0 0 5 3"
      className={`shrink-0 inline-block align-middle overflow-hidden ${className}`}
      aria-label="Deutschland"
    >
      <rect width="5" height="1" y="0" fill="#000000" />
      <rect width="5" height="1" y="1" fill="#DD0000" />
      <rect width="5" height="1" y="2" fill="#FFCE00" />
    </svg>
  </span>
);
