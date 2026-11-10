import React, { useRef, useState } from 'react';
import { Popover } from '@base-ui/react/popover';
import { cn } from './cn';
import { POPUP } from './popup';
import { Dialog } from './Dialog';

const MARK =
  'inline-flex items-center justify-center w-6 h-6 -my-1 shrink-0 align-middle ' +
  'text-caption font-semibold leading-none text-ink-subtle ' +
  'transition-colors duration-120 ease-quiet hover:text-ink-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent';

export interface HelpProps {
  /** Kurze Erklärung ohne Fachjargon, Quellen oder Bedienelemente. */
  summary: string;
  /** Ausführliche Erklärung für den Dialog. Normen werden dort verlinkt. */
  children?: React.ReactNode;
  label: string;
  /** Öffnet einen eigenen Detaildialog, etwa mit einer Berechnungstabelle. */
  onMore?: () => void;
  className?: string;
}

/**
 * Das Fragezeichen öffnet ein kleines Popover: der Kurztext und, wenn es
 * Details gibt, der Knopf „Mehr erfahren“ zum Detaildialog (§15.2). Es geht
 * beim Überstreichen, beim Tastaturfokus und beim Klick auf. Über den Fokus
 * geöffnet, bleibt der Fokus am Fragezeichen; Enter führt zu „Mehr erfahren“.
 */
export const Help: React.FC<HelpProps> = ({ summary, children, label, onMore, className }) => {
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const mark = useRef<HTMLButtonElement>(null);
  const popup = useRef<HTMLDivElement>(null);
  const more = useRef<HTMLButtonElement>(null);
  // Kehrt der Fokus aus Popover oder Detaildialog zurück, bleibt das Popover zu.
  const returning = useRef(false);
  // Über den Fokus geöffnet: der Fokus bleibt am Fragezeichen.
  const openedByFocus = useRef(false);
  const hasDetails = Boolean(children || onMore);

  return (
    <span className={cn('inline-flex items-center shrink-0 align-middle', className)}>
      <Popover.Root
        open={popoverOpen}
        onOpenChange={(next) => {
          if (!next && popup.current?.contains(document.activeElement)) returning.current = true;
          setPopoverOpen(next);
        }}
      >
        <Popover.Trigger
          ref={mark}
          aria-label={label}
          className={MARK}
          openOnHover
          delay={300}
          closeDelay={200}
          onFocus={(event) => {
            if (returning.current) {
              returning.current = false;
            } else if (!popoverOpen && event.currentTarget.matches(':focus-visible')) {
              openedByFocus.current = true;
              setPopoverOpen(true);
            }
          }}
          onBlur={() => {
            returning.current = false;
          }}
          onKeyDown={(event) => {
            // Steht das Popover schon, schließt Enter es nicht, sondern führt
            // zu „Mehr erfahren“.
            if (popoverOpen && event.key === 'Enter' && more.current) {
              event.preventDefault();
              more.current.focus();
            }
          }}
        >
          ?
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Positioner sideOffset={6} className="z-50">
            <Popover.Popup
              ref={popup}
              aria-label={label}
              initialFocus={(openType) => {
                const keep = openedByFocus.current;
                openedByFocus.current = false;
                return !keep && openType === 'keyboard';
              }}
              className={cn(POPUP, 'px-3 py-2 max-w-[300px] text-left text-body font-normal text-ink-muted')}
            >
              <p>{summary}</p>
              {hasDetails && (
                <button
                  ref={more}
                  type="button"
                  className="mt-2 text-caption font-normal text-ink-subtle underline underline-offset-2 hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent"
                  aria-label={`Mehr erfahren: ${label}`}
                  aria-haspopup="dialog"
                  onClick={() => {
                    setPopoverOpen(false);
                    if (onMore) onMore();
                    else setDetailsOpen(true);
                  }}
                >
                  Mehr erfahren
                </button>
              )}
            </Popover.Popup>
          </Popover.Positioner>
        </Popover.Portal>
      </Popover.Root>
      {!onMore && hasDetails && (
        <Dialog
          open={detailsOpen}
          onOpenChange={(next) => {
            if (!next) returning.current = true;
            setDetailsOpen(next);
          }}
          title={label}
          returnFocus={mark}
        >
          <div className="text-body font-normal text-ink-muted space-y-3">
            <p>{summary}</p>
            <div>{children}</div>
          </div>
        </Dialog>
      )}
    </span>
  );
};
