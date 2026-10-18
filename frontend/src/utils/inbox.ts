import type { InboxAdded, InboxInfo, InboxKind } from '../types';

/**
 * Der Eingang in Worten: wie viel dort liegt, seit wann, und was aus einer
 * ins Fenster gezogenen Datei geworden ist.
 */

/** Ab so vielen Tagen erinnert die Übersicht daran, Claude ablegen zu lassen. */
export const STALE_DAYS = 14;

/** Der Name des Eingangs im Datenordner, wie `domain.DefaultInboxDir`. */
export const DEFAULT_INBOX_DIR = 'Eingang';

/** Der Satz, mit dem der Mensch Claude den Eingang abarbeiten lässt. */
export const INBOX_PROMPT = 'Buche die Belege im Eingang.';

export const INBOX_KIND_LABELS: Record<InboxKind, string> = {
  beleg: 'Beleg',
  e_rechnung: 'E-Rechnung',
  kontoauszug: 'Kontoauszug',
  nicht_unterstuetzt: 'Nicht lesbar',
};

function files(count: number): string {
  return count === 1 ? '1 Datei' : `${count} Dateien`;
}

/** „seit heute“, „seit gestern“, „seit 19 Tagen“. */
export function sinceDays(days: number): string {
  if (days <= 0) return 'seit heute';
  if (days === 1) return 'seit gestern';
  return `seit ${days} Tagen`;
}

/** Die Kontextzeile des Eingangs: „3 Dateien · die älteste seit 19 Tagen“. */
export function inboxContext(info: InboxInfo): string {
  if (info.count === 0) return 'Leer';
  if (info.count === 1) return `1 Datei · ${sinceDays(info.oldestDays)}`;
  return `${files(info.count)} · die älteste ${sinceDays(info.oldestDays)}`;
}

/** Liegt die älteste Datei so lange, dass die Übersicht daran erinnert? */
export function inboxStale(info: InboxInfo): boolean {
  return info.count > 0 && info.oldestDays >= STALE_DAYS;
}

/**
 * Der Eingang, solange niemand einen anderen wählt: `<Ablage>/Eingang`, mit
 * dem Trennzeichen, das der Pfad der Ablage schon hat.
 */
export function defaultInbox(dataDir: string): string {
  const trimmed = dataDir.trim().replace(/[\\/]+$/, '');
  if (!trimmed) return '';
  const separator = trimmed.includes('\\') && !trimmed.includes('/') ? '\\' : '/';
  return `${trimmed}${separator}${DEFAULT_INBOX_DIR}`;
}

export interface DropFeedback {
  /** Wie viele Dateien jetzt im Eingang liegen. */
  added?: string;
  /** Welche Dateien nicht übernommen wurden, mit Grund. */
  skipped?: string;
  error?: string;
}

/** Die kurze Rückmeldung, nachdem Dateien ins Fenster gezogen wurden. */
export function dropFeedback(result: InboxAdded): DropFeedback {
  const out: DropFeedback = {};
  if (result.error) out.error = `Der Eingang hat die Dateien nicht angenommen: ${result.error}`;
  const names = result.names ?? [];
  if (names.length === 1) out.added = `${names[0]} liegt jetzt im Eingang.`;
  else if (names.length > 1) out.added = `${files(names.length)} liegen jetzt im Eingang.`;
  const skipped = result.skipped ?? [];
  if (skipped.length > 0) {
    out.skipped = `Nicht übernommen: ${skipped.map((s) => `${s.name} – ${s.reason}`).join('; ')}`;
  }
  return out;
}
