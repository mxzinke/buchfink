import React from 'react';
import { FilterX, Inbox } from 'lucide-react';
import { cn } from './cn';

export interface EmptyStateProps {
  /**
   * `leer` heißt: noch nichts erfasst. `gefiltert` heißt: es gibt Daten, der
   * Filter greift nur nicht. Wer die beiden verwechselt, lässt Nutzer glauben,
   * ihre Daten seien weg (§13).
   */
  variant?: 'leer' | 'gefiltert';
  icon?: React.ReactNode;
  title: string;
  /** Ein Satz. Mehr ist hier nicht vorgesehen (§15.1). */
  description?: string;
  /** Die eine Aktion, die weiterhilft. */
  action?: React.ReactNode;
  className?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  variant = 'leer',
  icon,
  title,
  description,
  action,
  className,
}) => {
  const fallback =
    variant === 'gefiltert' ? (
      <FilterX className="w-6 h-6" strokeWidth={1.5} />
    ) : (
      <Inbox className="w-6 h-6" strokeWidth={1.5} />
    );

  return (
    <div
      className={cn(
        'rounded-card border border-line bg-surface px-6 py-12 text-center',
        className,
      )}
    >
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-control bg-accent-soft text-accent-text">
        {icon ?? fallback}
      </div>
      <h3 className="text-heading text-ink mt-4">{title}</h3>
      {description && (
        <p className="text-body text-ink-muted mt-1 mx-auto max-w-md">{description}</p>
      )}
      {action && <div className="mt-5 flex justify-center gap-2">{action}</div>}
    </div>
  );
};
