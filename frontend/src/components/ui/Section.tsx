import React from 'react';
import { cn } from './cn';

export interface PageHeaderProps {
  title: string;
  /** Eine Zeile, höchstens 60 Zeichen (§15.1). */
  context?: string;
  action?: React.ReactNode;
  className?: string;
}

/** Jede Ansicht beginnt gleich: Titel, eine Zeile Kontext, rechts die
 *  Primäraktion, darunter eine Haarlinie. Kein Icon, kein Kasten (§12). */
export const PageHeader: React.FC<PageHeaderProps> = ({ title, context, action, className }) => (
  <header
    className={cn('flex items-start justify-between gap-4 pb-4 border-b border-line', className)}
  >
    <div className="min-w-0">
      <h1 className="text-display text-ink">{title}</h1>
      {context && <p className="text-caption text-ink-subtle mt-1 truncate">{context}</p>}
    </div>
    {action && <div className="shrink-0">{action}</div>}
  </header>
);

export interface SectionProps {
  title?: string;
  context?: string;
  action?: React.ReactNode;
  /** Der erste Abschnitt einer Ansicht braucht keine Linie, er steht schon
   *  unter dem Seitenkopf (§6.4). */
  divider?: boolean;
  className?: string;
  children: React.ReactNode;
}

/** Der Ersatz für die Karte: Überschrift, Abstand, Haarlinie. */
export const Section: React.FC<SectionProps> = ({
  title,
  context,
  action,
  divider = true,
  className,
  children,
}) => (
  <section className={cn(divider && 'mt-8 pt-8 border-t border-line', className)}>
    {(title || action) && (
      <div className="flex items-start justify-between gap-4 mb-5">
        <div className="min-w-0">
          {title && <h2 className="text-heading text-ink">{title}</h2>}
          {context && <p className="text-caption text-ink-subtle mt-1">{context}</p>}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </div>
    )}
    {children}
  </section>
);
