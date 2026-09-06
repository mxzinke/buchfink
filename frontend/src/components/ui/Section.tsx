import React from 'react';
import { cn } from './cn';
import { HelpPopover } from './Help';

/**
 * Erklärzeichen an einer Überschrift.
 *
 * Es steht hinter der Beschriftung, nie davor (§15.2) — und damit hier und
 * nicht in der Aktionsspalte. Dort landete es am rechten Rand neben den
 * Knöpfen, also an einer Stelle, an der es aussieht wie eine weitere Aktion und
 * niemand die Erklärung zur Überschrift vermutet.
 */
const TitleHelp: React.FC<{
  title?: string;
  explain?: React.ReactNode;
  onMore?: () => void;
}> = ({ title, explain, onMore }) => {
  if (!explain) return null;
  return (
    <HelpPopover label={`Erklärung zu ${title ?? 'diesem Abschnitt'}`} onMore={onMore}>
      {explain}
    </HelpPopover>
  );
};

export interface PageHeaderProps {
  title: string;
  /** Eine Zeile, höchstens 60 Zeichen (§15.1). */
  context?: string;
  /** Ein bis drei Sätze hinter dem Erklärzeichen am Titel (§15.2). */
  explain?: React.ReactNode;
  /** Öffnet die dritte Stufe: „Mehr dazu" im Popover (§15.2). */
  onMore?: () => void;
  action?: React.ReactNode;
  className?: string;
}

/** Jede Ansicht beginnt gleich: Titel, eine Zeile Kontext, rechts die
 *  Primäraktion, darunter eine Haarlinie. Kein Icon, kein Kasten (§12). */
export const PageHeader: React.FC<PageHeaderProps> = ({
  title,
  context,
  explain,
  onMore,
  action,
  className,
}) => (
  <header
    className={cn('flex items-start justify-between gap-4 pb-4 border-b border-line', className)}
  >
    <div className="min-w-0">
      <h1 className="flex items-center text-display text-ink">
        {title}
        <TitleHelp title={title} explain={explain} onMore={onMore} />
      </h1>
      {context && <p className="text-caption text-ink-subtle mt-1 truncate">{context}</p>}
    </div>
    {action && <div className="shrink-0">{action}</div>}
  </header>
);

export interface SectionProps {
  /** Anker, über den ein Verweis auf derselben Seite den Abschnitt anspringt. */
  id?: string;
  title?: string;
  context?: string;
  /** Ein bis drei Sätze hinter dem Erklärzeichen am Titel (§15.2). */
  explain?: React.ReactNode;
  /** Öffnet die dritte Stufe: „Mehr dazu" im Popover (§15.2). */
  onMore?: () => void;
  action?: React.ReactNode;
  /** Der erste Abschnitt einer Ansicht braucht keine Linie, er steht schon
   *  unter dem Seitenkopf (§6.4). */
  divider?: boolean;
  className?: string;
  children: React.ReactNode;
}

/** Der Ersatz für die Karte: Überschrift, Abstand, Haarlinie. */
export const Section: React.FC<SectionProps> = ({
  id,
  title,
  context,
  explain,
  onMore,
  action,
  divider = true,
  className,
  children,
}) => (
  <section id={id} className={cn(divider && 'mt-8 pt-8 border-t border-line', className)}>
    {(title || action) && (
      <div className="flex items-start justify-between gap-4 mb-5">
        <div className="min-w-0">
          {title && (
            <h2 className="flex items-center text-heading text-ink">
              {title}
              <TitleHelp title={title} explain={explain} onMore={onMore} />
            </h2>
          )}
          {context && <p className="text-caption text-ink-subtle mt-1">{context}</p>}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </div>
    )}
    {children}
  </section>
);
