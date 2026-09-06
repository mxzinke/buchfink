import React from 'react';
import { Popover } from '@base-ui/react/popover';
import { cn } from './cn';
import { POPUP } from './popup';

/**
 * Das Erklärzeichen aus §15.2 — ein Fragezeichen hinter der Beschriftung, nie
 * davor. Ausgelöst wird bewusst: keine Tour, kein Popover beim ersten Besuch.
 *
 * Es gibt genau eine Erklärstufe an diesem Zeichen. Bis Welle 9 gab es zwei —
 * einen dunklen Tooltip für einen Satz und ein helles Popover für drei —, beide
 * hinter demselben Fragezeichen und mit demselben Aussehen. Von außen war nicht
 * zu erkennen, welche der beiden man vor sich hatte: Manche gingen beim
 * Überstreichen auf, andere erst auf Klick, manche waren dunkel, andere hell,
 * und in einer Reihe von Feldern standen beide nebeneinander. Ein Unterschied,
 * der nur im Code besteht, ist keiner — er sieht aus wie ein Fehler.
 *
 * Geblieben ist das Popover: es trägt den einen Satz genauso wie die drei und
 * darf einen Verweis in die dritte Stufe enthalten, den ein Tooltip nicht
 * tragen kann.
 *
 * Das Klickfeld ist 24 px hoch, die Zeile darunter aber oft nur 16. Deshalb der
 * negative Rand: Das Zeichen darf die Zeile, in der es steht, nicht auseinander
 * ziehen — sonst stehen zwei Felder nebeneinander verschieden hoch, je nachdem,
 * ob eines von beiden eine Erklärung hat.
 */
const MARK =
  'inline-flex items-center justify-center w-6 h-6 -my-1 shrink-0 align-middle ' +
  'text-caption font-semibold leading-none text-ink-faint ' +
  'transition-colors duration-120 ease-quiet hover:text-ink-muted data-[popup-open]:text-ink-muted';

/**
 * Das Popover geht beim Überstreichen auf, ruhig: 300 ms bis es kommt, 200 ms
 * bis es wieder geht. Der Klick bleibt daneben bestehen — er ist der Weg mit
 * der Tastatur und auf dem Touchgerät, wo es kein Hover gibt.
 */
const POPOVER_HOVER = { openOnHover: true, delay: 300, closeDelay: 200 } as const;

export interface HelpPopoverProps {
  /** Ein bis drei Sätze. Was länger ist, gehört in die dritte Stufe. */
  children: React.ReactNode;
  /** Beschriftung für Screenreader, etwa "Erklärung zu Einnahmen". */
  label: string;
  /** Sprung in die dritte Stufe. Die Beschriftung ist immer "Mehr dazu". */
  onMore?: () => void;
  className?: string;
}

export const HelpPopover: React.FC<HelpPopoverProps> = ({
  children,
  label,
  onMore,
  className,
}) => (
  <Popover.Root>
    <Popover.Trigger {...POPOVER_HOVER} aria-label={label} className={cn(MARK, className)}>
      ?
    </Popover.Trigger>
    <Popover.Portal>
      <Popover.Positioner sideOffset={6} className="z-50">
        <Popover.Popup className={cn(POPUP, 'p-4 max-w-[340px] text-left')}>
          <Popover.Description className="text-body text-ink-muted">
            {children}
          </Popover.Description>
          {onMore && (
            <Popover.Close
              onClick={onMore}
              className="mt-3 text-label font-semibold text-accent-text hover:text-accent transition-colors duration-120 ease-quiet"
            >
              Mehr dazu
            </Popover.Close>
          )}
        </Popover.Popup>
      </Popover.Positioner>
    </Popover.Portal>
  </Popover.Root>
);

export interface InfoPopoverProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

/** Popover an einem eigenen Auslöser, etwa einem Button in einer Tabellenzeile. */
export const InfoPopover: React.FC<InfoPopoverProps> = ({ trigger, children, className }) => (
  <Popover.Root>
    <Popover.Trigger render={trigger as React.ReactElement} />
    <Popover.Portal>
      <Popover.Positioner sideOffset={6} className="z-50">
        <Popover.Popup className={cn(POPUP, 'p-4 max-w-[340px]', className)}>
          {children}
        </Popover.Popup>
      </Popover.Positioner>
    </Popover.Portal>
  </Popover.Root>
);
