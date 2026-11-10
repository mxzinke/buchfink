import type { InboxAdded, InboxInfo, InboxKind } from '../types';

/**
 * Der Eingang in Worten: wie viel dort liegt, wie alt die älteste Datei ist,
 * und was aus einer ins Fenster gezogenen Datei geworden ist. Das Alter ist
 * das der Datei (Änderungsdatum), nicht die Zeit im Eingang: wer eine alte
 * Rechnung aus den Downloads hineinzieht, bringt ihr Datum mit.
 */

/** Ist die älteste Datei so viele Tage alt, erinnert die Übersicht daran, Claude ablegen zu lassen. */
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

/** „von heute“, „von gestern“, „19 Tage alt“. */
export function ageText(days: number): string {
  if (days <= 0) return 'von heute';
  if (days === 1) return 'von gestern';
  return `${days} Tage alt`;
}

/** Die Kontextzeile des Eingangs: „3 Dateien · die älteste 19 Tage alt“. */
export function inboxContext(info: InboxInfo): string {
  if (info.count === 0) return 'Leer';
  if (info.count === 1) return `1 Datei · ${ageText(info.oldestDays)}`;
  return `${files(info.count)} · die älteste ${ageText(info.oldestDays)}`;
}

/** Ist die älteste Datei so alt, dass die Übersicht daran erinnert? */
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
