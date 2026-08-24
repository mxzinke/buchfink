import React from 'react';
import { cn } from './cn';

/**
 * Kennzahlen sind keine Kacheln. Sie stehen in einer Reihe direkt auf dem
 * Papier, getrennt durch senkrechte Haarlinien (§10).
 *
 * Ab fünf Kennzahlen bricht die Reihe um. Die Linien entfallen dann zugunsten
 * von Abstand, weil sie über mehrere Zeilen nur noch Gitter wären.
 */
const ROW =
  'flex [&>*]:flex-1 [&>*]:px-6 [&>*]:border-l [&>*]:border-line ' +
  '[&>*:first-child]:pl-0 [&>*:first-child]:border-l-0 [&>*:last-child]:pr-0';

const WRAPPED = 'grid grid-cols-2 lg:grid-cols-3 gap-x-8 gap-y-6';

export const StatRow: React.FC<{ className?: string; children: React.ReactNode }> = ({
  className,
  children,
}) => {
  const wrapped = React.Children.count(children) > 4;
  return <div className={cn(wrapped ? WRAPPED : ROW, className)}>{children}</div>;
};

export interface StatProps {
  label: string;
  value: React.ReactNode;
  /** Eine Zeile Herkunft, etwa "Geschäftskonto 1800". */
  context?: string;
  /** Nur das Ergebnis wird gefärbt, nie eine laufende Zahl (§3.4). */
  tone?: 'neutral' | 'positive' | 'negative';
}

const TONE = {
  neutral: 'text-ink',
  positive: 'text-positive-text',
  negative: 'text-negative-text',
};

export const Stat: React.FC<StatProps> = ({ label, value, context, tone = 'neutral' }) => (
  <div className="min-w-0">
    <div className="text-caption text-ink-subtle">{label}</div>
    <div className={cn('text-display num mt-1 truncate', TONE[tone])}>{value}</div>
    {context && <div className="text-caption text-ink-subtle mt-0.5 truncate">{context}</div>}
  </div>
);
