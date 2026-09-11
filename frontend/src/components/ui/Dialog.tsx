import React, { useState } from 'react';
import { Dialog as Base } from '@base-ui/react/dialog';
import { AlertDialog as BaseAlert } from '@base-ui/react/alert-dialog';
import { X } from 'lucide-react';
import { cn } from './cn';
import { Button } from './Button';
import { legalText } from './LegalText';
import { BACKDROP, POPUP } from './popup';

const PANEL =
  'fixed left-1/2 top-1/2 z-50 w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2 ' +
  'max-h-[calc(100vh-4rem)] overflow-y-auto shadow-dialog';

export interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Aktionen rechts, Sekundär links von Primär. */
  footer?: React.ReactNode;
  /** Tailwind-Breite, etwa `max-w-xl`. */
  width?: string;
  /**
   * Steht im Dialog eine begonnene Eingabe? Dann fragt das Schließen nach
   * (§8.7). Der Dialog kennt den Inhalt seiner Felder nicht; die Ansicht
   * meldet ihn hier, und die Rückfrage steht dafür nur an einer Stelle.
   */
  dirty?: boolean;
  children: React.ReactNode;
}

/**
 * Für eine abgeschlossene Eingabe. Was länger als ein Bildschirm ist, etwa die
 * Rechnungserfassung oder der Beleg-Buchungsflow, gehört in eine eigene Ansicht.
 *
 * Fokusfang, Escape und Fokusrückgabe an das auslösende Element übernimmt Base
 * UI (§8.8). Eine begonnene Eingabe geht beim Schließen nicht verloren: mit
 * `dirty` fragt der Dialog nach (§8.7).
 */
export const Dialog: React.FC<DialogProps> = ({
  open,
  onOpenChange,
  title,
  footer,
  width = 'max-w-2xl',
  dirty = false,
  children,
}) => {
  const [askDiscard, setAskDiscard] = useState(false);

  // Escape, der Klick daneben und das Kreuz laufen alle über `onOpenChange`.
  // Die Rückfrage sitzt deshalb hier und nicht an den drei Bedienwegen.
  const handleOpenChange = (next: boolean) => {
    if (!next && dirty) {
      setAskDiscard(true);
      return;
    }
    onOpenChange(next);
  };

  return (
    <Base.Root open={open} onOpenChange={handleOpenChange}>
      <Base.Portal>
        <Base.Backdrop className={BACKDROP} />
        <Base.Popup className={cn(POPUP, PANEL, width)}>
          <div className="flex items-center justify-between gap-4 px-6 py-4 border-b border-line">
            <Base.Title className="text-heading text-ink">{title}</Base.Title>
            <Base.Close
              render={
                <Button variant="quiet" size="sm" iconOnly title="Schließen" aria-label="Dialog schließen">
                  <X className="w-4 h-4" strokeWidth={1.5} />
                </Button>
              }
            />
          </div>

          <div className="px-6 py-5">{legalText(children)}</div>

          {footer && (
            <div className="flex flex-wrap justify-end gap-2 px-6 py-4 border-t border-line">
              {footer}
            </div>
          )}
        </Base.Popup>
      </Base.Portal>
      {/* Die Rückfrage steht innerhalb des Dialogs, damit Base UI sie als
          verschachtelt erkennt: als Geschwister nähme der Fokusfang des
          äußeren Dialogs ihr die Bedienbarkeit. Verwerfen ist unumkehrbar, die
          begonnene Eingabe ist danach fort (§8.2). */}
      <ConfirmDialog
        open={askDiscard}
        onOpenChange={setAskDiscard}
        title="Eingabe verwerfen?"
        description="Die begonnene Eingabe geht verloren. Sie lässt sich nicht wiederherstellen."
        confirmLabel="Verwerfen"
        destructive
        onConfirm={() => onOpenChange(false)}
      />
    </Base.Root>
  );
};

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Benennt die Folge, nicht die Aktion (§8.2). Zwei Sätze genügen. */
  description: string;
  confirmLabel: string;
  onConfirm: () => void;
  /** Storno und Löschen. Sonst bleibt die Bestätigung in Tinte. */
  destructive?: boolean;
}

/**
 * Nur für unumkehrbare Schritte: Festschreiben, Periode abschließen, Mandant
 * löschen. Was rückgängig gemacht werden kann, wird ohne Rückfrage ausgeführt
 * und bekommt einen Toast mit Rückgängig (§8.2).
 *
 * Anders als der Dialog lässt er sich nicht durch einen Klick daneben
 * schließen, sondern nur über eine der beiden Antworten.
 */
export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  onConfirm,
  destructive = false,
}) => (
  <BaseAlert.Root open={open} onOpenChange={onOpenChange}>
    <BaseAlert.Portal>
      <BaseAlert.Backdrop className={BACKDROP} />
      <BaseAlert.Popup className={cn(POPUP, PANEL, 'max-w-md')}>
        <div className="px-6 py-5">
          <BaseAlert.Title className="text-heading text-ink">{title}</BaseAlert.Title>
          <BaseAlert.Description className="text-body text-ink-muted mt-2">
            {description}
          </BaseAlert.Description>
        </div>
        <div className="flex justify-end gap-2 px-6 py-4 border-t border-line">
          <BaseAlert.Close render={<Button variant="secondary">Abbrechen</Button>} />
          <Button
            variant={destructive ? 'danger' : 'primary'}
            onClick={() => {
              onOpenChange(false);
              onConfirm();
            }}
          >
            {confirmLabel}
          </Button>
        </div>
      </BaseAlert.Popup>
    </BaseAlert.Portal>
  </BaseAlert.Root>
);
