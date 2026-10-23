import React from 'react';
import { Dialog as Base } from '@base-ui/react/dialog';
import { X } from 'lucide-react';
import { cn } from './cn';
import { Button } from './Button';
import { legalText } from './LegalText';
import { BACKDROP } from './popup';

const WIDTH = {
  md: 'max-w-[480px]',
  lg: 'max-w-[720px]',
};

/** Overlay am rechten Rand; öffnet mit 4 px Versatz, schließt nur über die Deckkraft (§7). */
const PANEL =
  'fixed inset-y-0 right-0 z-50 flex w-full flex-col border-l border-line bg-surface shadow-dialog ' +
  'outline-none transition-[opacity,transform] duration-180 ease-quiet ' +
  'data-[starting-style]:opacity-0 data-[starting-style]:translate-x-1 ' +
  'data-[ending-style]:opacity-0 data-[ending-style]:duration-120';

export interface DetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Eine Zeile unter dem Titel, etwa Datum und Betrag. */
  context?: string;
  /** Neben dem Titel, etwa ein StatusBadge. */
  meta?: React.ReactNode;
  /** Aktionen am Fuß, Sekundär links von Primär. */
  footer?: React.ReactNode;
  width?: keyof typeof WIDTH;
  children: React.ReactNode;
}

/**
 * Die Seitenleiste für die Einzelheiten eines Listeneintrags.
 *
 * Die Liste zeigt Zustand und Kennzahl, die Seitenleiste den Rest: Buchungssatz,
 * Herkunft, Bearbeiter, Verweise. Geöffnet wird sie über die Kennung im Ziel
 * (`Route.id`), damit ein Verweis aus einer anderen Ansicht direkt auf dem
 * Eintrag landet und Zurück sie wieder schließt.
 *
 * Fokusfang, Escape und Fokusrückgabe kommen von Base UI (§8.8).
 */
export const DetailPanel: React.FC<DetailPanelProps> = ({
  open,
  onOpenChange,
  title,
  context,
  meta,
  footer,
  width = 'md',
  children,
}) => (
  <Base.Root open={open} onOpenChange={onOpenChange}>
    <Base.Portal>
      <Base.Backdrop className={BACKDROP} />
      <Base.Popup className={cn(PANEL, WIDTH[width])}>
        <div className="flex items-start justify-between gap-4 px-6 py-4 border-b border-line">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <Base.Title className="text-heading text-ink">{title}</Base.Title>
              {meta}
            </div>
            {context && <p className="text-caption text-ink-subtle mt-1 truncate">{context}</p>}
          </div>
          <Base.Close
            render={
              <Button variant="quiet" size="sm" iconOnly title="Schließen" aria-label="Einzelheiten schließen">
                <X className="w-4 h-4" strokeWidth={1.5} />
              </Button>
            }
          />
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-5">{legalText(children)}</div>

        {footer && (
          <div className="flex flex-wrap justify-end gap-2 px-6 py-4 border-t border-line">
            {footer}
          </div>
        )}
      </Base.Popup>
    </Base.Portal>
  </Base.Root>
);

export interface DetailListProps {
  items: { label: string; value: React.ReactNode; numeric?: boolean }[];
  className?: string;
}

/** Bezeichnung und Wert untereinander ausgerichtet, für Seitenleiste und aufgeklappte Zeile. */
export const DetailList: React.FC<DetailListProps> = ({ items, className }) => (
  <dl className={cn('grid grid-cols-[minmax(8rem,auto)_1fr] gap-x-6 gap-y-2 text-body', className)}>
    {items.map((item) => (
      <React.Fragment key={item.label}>
        <dt className="text-ink-subtle">{item.label}</dt>
        <dd className={cn('text-ink min-w-0 break-words', item.numeric && 'num')}>{item.value}</dd>
      </React.Fragment>
    ))}
  </dl>
);
