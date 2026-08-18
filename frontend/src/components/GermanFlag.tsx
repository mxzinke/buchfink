import React from 'react';

interface GermanFlagProps {
  className?: string;
}

export const GermanFlag: React.FC<GermanFlagProps> = ({ className = 'w-4 h-3' }) => {
  return (
    <span title="Entwickelt für deutsches Steuerrecht & GoBD" className="inline-flex items-center">
      <svg
        viewBox="0 0 5 3"
        className={`rounded-xs overflow-hidden shadow-xs shrink-0 inline-block align-middle ${className}`}
        aria-label="Deutschland / German Accounting"
      >
        <rect width="5" height="1" y="0" fill="#1c1917" />
        <rect width="5" height="1" y="1" fill="#dc2626" />
        <rect width="5" height="1" y="2" fill="#f59e0b" />
      </svg>
    </span>
  );
};
