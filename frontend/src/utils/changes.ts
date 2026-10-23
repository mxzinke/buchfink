import type { ChangeEvent, ChangeKind } from '../types';

/**
 * Die Regeln, nach denen die Oberfläche auf Änderungen im Backend neu lädt.
 * Das Backend meldet sie als `buchfink:geaendert`, gebündelt je Mandant und
 * höchstens alle 250 ms (internal/wailsbridge/changes.go).
 */

/** Gilt ein Ereignis dem geöffneten Mandanten? Ein anderer Mandant lädt nichts neu. */
export function forTenant(event: ChangeEvent | null | undefined, tenantId: string | undefined): boolean {
  return Boolean(event && tenantId && event.tenantId === tenantId);
}

/**
 * Betrifft eine Änderung, was eine Ansicht zeigt? Ohne eigene Arten zeigt die
 * Ansicht alles; ohne gemeldete Art ist unbekannt, was sich geändert hat, und
 * sie lädt ebenfalls neu.
 */
export function concerns(changed: readonly ChangeKind[], wanted?: readonly ChangeKind[]): boolean {
  if (!wanted || wanted.length === 0 || changed.length === 0) return true;
  return wanted.some((kind) => changed.includes(kind));
}

/**
 * Wie lange bis zum Neuladen: `settle` nach dem letzten Ereignis, damit eine
 * Folge von Buchungen einmal lädt, aber höchstens `max` nach dem ersten. Ein
 * Agent, der zwanzig Belege hintereinander ablegt, ließe die Ansicht sonst
 * stehen, bis er fertig ist.
 */
export function reloadDelay(firstAt: number, now: number, settle: number, max: number): number {
  return Math.max(0, Math.min(settle, firstAt + max - now));
}

/**
 * Nummeriert die Ladevorgänge einer Ansicht. `begin` liefert für jeden eine
 * Prüfung, ob er noch der jüngste ist. Antworten kommen nicht in der
 * Reihenfolge der Anfragen zurück: ein Neuladen nach einem Ereignis, das vor
 * einem Wechsel des Geschäftsjahres begann, darf den Stand des neuen Jahres
 * nicht überschreiben.
 */
export function requestCounter(): { begin: () => () => boolean } {
  let latest = 0;
  return {
    begin() {
      const id = ++latest;
      return () => id === latest;
    },
  };
}
