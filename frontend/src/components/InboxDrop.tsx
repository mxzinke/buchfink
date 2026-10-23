import React, { useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { Inbox } from 'lucide-react';
import type { InboxAdded } from '../types';
import { normalizeInboxAdded } from '../services/api';
import { EVENT_INBOX_ADDED } from '../live';
import { dropFeedback, type DropFeedback } from '../utils/inbox';
import { toast } from './ui';

/**
 * Das Attribut, an dem die Wails-Laufzeit eine Ablagezone erkennt. Ein Drop
 * außerhalb eines solchen Elements bleibt ohne Wirkung; über dem Element setzt
 * die Laufzeit während des Ziehens die Klasse `file-drop-target-active`.
 */
export const DROP_TARGET = { 'data-file-drop-target': 'eingang' } as const;

/**
 * Eine Meldung für das Ergebnis: was im Eingang liegt, darunter, was nicht
 * übernommen wurde. Zwei getrennte Meldungen lägen übereinander, und die
 * untere wäre nicht zu lesen.
 */
export function showDropFeedback(feedback: DropFeedback): void {
  if (feedback.error) toast.error(feedback.error);
  if (feedback.added) toast.success(feedback.added, feedback.skipped);
  else if (feedback.skipped) toast.error(feedback.skipped);
}

/**
 * Die Ablagezone über dem Arbeitsbereich. Sie steht als Kind des Elements mit
 * `DROP_TARGET` und erscheint nur, solange Dateien über das Fenster gezogen
 * werden; die Regel dafür steht in index.css und hängt an der Klasse der
 * Laufzeit. Kopiert wird im Backend (`wailsbridge.AttachFileDrop`), das
 * Ergebnis kommt als Ereignis zurück und erscheint hier als kurze Meldung.
 */
export const InboxDropZone: React.FC = () => {
  useEffect(
    () =>
      Events.On(EVENT_INBOX_ADDED, (event) => {
        showDropFeedback(dropFeedback(normalizeInboxAdded(event.data as InboxAdded)));
      }),
    [],
  );

  return (
    // Über der Navigation (z-50), damit die Zone das ganze Fenster zeigt.
    <div className="drop-zone fixed inset-0 z-[60] items-center justify-center bg-paper/85 p-6" aria-hidden="true">
      <div className="flex h-full w-full flex-col items-center justify-center gap-2 rounded-overlay border-2 border-dashed border-accent bg-accent-soft/70 text-center">
        <Inbox className="w-7 h-7 text-accent-text" strokeWidth={1.5} />
        <p className="text-heading text-ink">In den Eingang legen</p>
        <p className="max-w-sm text-body text-ink-muted">
          Belege, E-Rechnungen und Kontoauszüge. Claude legt sie ab, wenn Sie es sagen.
        </p>
      </div>
    </div>
  );
};
