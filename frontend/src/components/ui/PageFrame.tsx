import React, { createContext, useContext } from 'react';
import { cn } from './cn';
import { Help } from './Help';
import { TabPanel, Tabs } from './Tabs';

/** Breite und Rand des Arbeitsbereichs (§5). */
export const PAGE_CONTAINER = 'max-w-[1200px] mx-auto px-4 sm:px-8';

/** Für Ansichten mit Dokument und Erfassung nebeneinander, etwa die Belege (§11.6). */
const PAGE_CONTAINER_WIDE = 'max-w-[1440px] mx-auto px-4 sm:px-8';

export interface FrameTab<T extends string> {
  value: T;
  label: string;
  /** Zähler rechts der Beschriftung, etwa offene Belege. */
  count?: number;
  countTone?: 'neutral' | 'negative';
  countLabel?: string;
}

export interface PageFrameProps<T extends string> {
  title: string;
  /** Eine Zeile, höchstens 60 Zeichen (§15.1). */
  context?: string;
  /** Ein kurzer Satz für den Tooltip am Titel. */
  helpSummary?: string;
  /** Ausführliche Erklärung im Detaildialog. */
  explain?: React.ReactNode;
  /** Die Primäraktion des Bereichs, rechts im Kopf. */
  action?: React.ReactNode;
  /** Hinweis zwischen Kopf und Inhalt, etwa die Sperre des Geschäftsjahres. */
  notice?: React.ReactNode;
  tabs?: FrameTab<T>[];
  /** Der aktive Reiter. Ohne `tabs` ohne Bedeutung. */
  tab?: T;
  onTabChange?: (tab: T) => void;
  /** Breiter als 1200 px, für Beleg und Erfassung nebeneinander. */
  wide?: boolean;
  children: React.ReactNode;
}

const FrameContext = createContext(false);

/**
 * Steht eine Ansicht in einem Seitenrahmen? Dann trägt der Rahmen den
 * Seitentitel, und `PageHeader` stuft sich zur Überschrift des Reiters herab.
 */
export function useInFrame(): boolean {
  return useContext(FrameContext);
}

/**
 * Der Rahmen eines Bereichs: Titel, eine Zeile Kontext, Primäraktion, darunter
 * die Reiter und der Inhalt des aktiven Reiters.
 *
 * Gerendert wird nur der aktive Reiter. Der Rahmen kennt die Route nicht; der
 * Bereich übersetzt Reiterwechsel in eine Navigation, damit Zurück und Vor der
 * WebView auch zwischen Reitern wirken.
 */
export function PageFrame<T extends string>({
  title,
  context,
  helpSummary,
  explain,
  action,
  notice,
  tabs,
  tab,
  onTabChange,
  wide = false,
  children,
}: PageFrameProps<T>) {
  const hasTabs = Boolean(tabs && tabs.length > 0 && tab !== undefined);

  return (
    <div className={cn(wide ? PAGE_CONTAINER_WIDE : PAGE_CONTAINER, 'py-8')}>
      <header
        className={cn(
          'flex items-start justify-between gap-4 pb-4',
          !hasTabs && 'border-b border-line',
        )}
      >
        <div className="min-w-0">
          <h1 aria-label={title} className="flex flex-wrap items-center text-display text-ink">
            {title}
            {explain && helpSummary && (
              <Help summary={helpSummary} label={`Erklärung zu ${title}`}>
                {explain}
              </Help>
            )}
          </h1>
          {context && <p className="text-caption text-ink-subtle mt-1 truncate">{context}</p>}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </header>

      <FrameContext.Provider value>
        {hasTabs ? (
          <Tabs<T> items={tabs!} value={tab} onValueChange={(next) => onTabChange?.(next)}>
            {notice && <div className="pt-6">{notice}</div>}
            <TabPanel value={tab!}>{children}</TabPanel>
          </Tabs>
        ) : (
          <>
            {notice && <div className="pt-6">{notice}</div>}
            <div className="pt-6">{children}</div>
          </>
        )}
      </FrameContext.Provider>
    </div>
  );
}
