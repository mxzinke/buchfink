import type {
  ConnectedApp,
  EvidenceKind,
  Obligation,
  ObligationEvidenceSource,
  ObligationResponsibility,
  Task,
  TaskGroup,
  TaskList,
} from '../types';

/**
 * Was die Übersicht aus Aufgabenliste und Pflichtenkalender zusammenstellt.
 *
 * Welche Pflicht jetzt dran ist, entscheidet die Aufgabenliste des Backends
 * (verstrichene, die der nächsten dreißig Tage, die Festschreibung ab dem
 * 10. des Folgemonats). Der Kalender liefert dazu Stand, Vorbereitung und Ziel.
 */

/** Der Schlüssel einer Pflicht steht in der Aufgabenliste hinter diesem Präfix. */
export const OBLIGATION_TASK_PREFIX = 'pflicht.';

/** Eine Pflicht unter „Was jetzt zu tun ist“. */
export interface DutyRow {
  /** Schlüssel der Pflicht, etwa „ustva.2026-Q3“. */
  key: string;
  group: TaskGroup;
  title: string;
  dueDate?: string;
  /** Fehlt, wenn der Kalender sie nicht mehr als offen führt. */
  obligation?: Obligation;
}

export interface OverviewPlan {
  /** Pflichten, die der Mensch erledigt oder bestätigt; überfällige zuerst. */
  duties: DutyRow[];
  /** Die übrige Arbeit in den Büchern, in der Reihenfolge der Aufgabenliste. */
  work: Task[];
  /** Die nächste Pflicht, wenn gerade keine ansteht. */
  next?: Obligation;
}

const GROUP_ORDER: Record<TaskGroup, number> = { overdue: 0, open: 1, upcoming: 2 };

export function planOverview(tasks: TaskList | null, obligations: Obligation[]): OverviewPlan {
  const byKey = new Map(obligations.map((o) => [o.key, o]));
  const seen = new Set<string>();
  const duties: DutyRow[] = [];
  const work: Task[] = [];

  for (const group of ['overdue', 'open', 'upcoming'] as const) {
    for (const task of tasks?.[group] ?? []) {
      if (!task.key.startsWith(OBLIGATION_TASK_PREFIX)) {
        work.push(task);
        continue;
      }
      const key = task.key.slice(OBLIGATION_TASK_PREFIX.length);
      if (seen.has(key)) continue;
      seen.add(key);
      duties.push({ key, group, title: task.title, dueDate: task.dueDate, obligation: byKey.get(key) });
    }
  }

  // Eine vorbereitete Pflicht wartet auf den Menschen, auch wenn ihre Frist
  // noch weiter weg ist als dreißig Tage.
  for (const o of obligations) {
    if (o.status !== 'prepared' || seen.has(o.key)) continue;
    seen.add(o.key);
    duties.push({ key: o.key, group: 'open', title: o.title, dueDate: o.dueDate, obligation: o });
  }

  duties.sort(
    (a, b) =>
      GROUP_ORDER[a.group] - GROUP_ORDER[b.group] || (a.dueDate ?? '').localeCompare(b.dueDate ?? ''),
  );

  const next =
    duties.length === 0
      ? obligations.find((o) => o.status !== 'done' && o.status !== 'waived')
      : undefined;
  return { duties, work, next };
}

/**
 * Der Monat, dessen Abschluss ansteht („JJJJ-MM“): der älteste vergangene Monat
 * mit offener Festschreibung, sonst der Monat vor `today`.
 */
export function monthToClose(obligations: Obligation[], today: string): string {
  const open = obligations
    .filter((o) => o.kind === 'festschreibung' && o.periodTo && o.periodTo < today)
    .map((o) => o.periodKey)
    .sort();
  if (open.length > 0) return open[0];
  const [year, month] = today.split('-').map(Number);
  if (!year || !month) return '';
  const previous = month === 1 ? { y: year - 1, m: 12 } : { y: year, m: month - 1 };
  return `${previous.y}-${String(previous.m).padStart(2, '0')}`;
}

/**
 * Der Name der App hinter einem Erfassungsweg `agent:<kurzname>`. Kennt
 * Buchfink die App nicht mehr, steht der Kurzname da.
 */
export function appName(channel: string | undefined, apps: ConnectedApp[]): string {
  if (!channel || channel === 'ui') return 'Sie in der App';
  if (channel === 'system') return 'Programmlauf';
  const app = apps.find((a) => a.channel === channel);
  if (app) return app.name;
  return channel.startsWith('agent:') ? channel.slice('agent:'.length) : channel;
}

/** Haben nur Apps von Claude gebucht? Dann heißt der Abschnitt nach Claude. */
export function onlyClaude(channels: (string | undefined)[]): boolean {
  return channels.every((channel) => !channel || channel.startsWith('agent:claude'));
}

/** Der Weg, auf dem eine Pflicht erfüllt wird. Intern erledigte stehen ohne. */
export const CHANNEL_LABELS: Record<string, string> = {
  elster: 'Mein ELSTER',
  bzst: 'Bundeszentralamt für Steuern über Mein ELSTER',
  unternehmensregister: 'Unternehmensregister',
  transparenzregister: 'Transparenzregister',
  notar: 'Notar',
  gewerbeamt: 'Gewerbeamt',
  lohnanbieter: 'Lohnanbieter',
};

export const RESPONSIBILITY_LABELS: Record<ObligationResponsibility, string> = {
  agent_prepares: 'Claude bereitet vor, Sie geben ab',
  human_confirms: 'Sie',
  tax_advisor: 'Ihr Steuerberater',
};

export const EVIDENCE_SOURCE_LABELS: Record<ObligationEvidenceSource, string> = {
  data: 'Buchfink erkennt es an den Daten',
  human: 'Ihre Bestätigung mit Nachweis',
  either: 'Die Daten oder Ihre Bestätigung',
};

export const EVIDENCE_LABELS: Record<EvidenceKind, string> = {
  transfer_ticket: 'Transferticket',
  file_number: 'Aktenzeichen',
  order_number: 'Auftragsnummer',
  booking: 'Buchung',
  commitment: 'Festschreibung',
  resolution: 'Beschluss',
  status: 'Abschlussstand',
  confirmation: 'Bestätigung',
};

/** Die Nachweise, die der Mensch beim Bestätigen wählen kann. */
export const CONFIRMABLE_EVIDENCE: EvidenceKind[] = [
  'transfer_ticket',
  'file_number',
  'order_number',
  'confirmation',
];

/** Zu diesen Nachweisen gehört eine Nummer. */
export function needsReference(kind: EvidenceKind): boolean {
  return kind === 'transfer_ticket' || kind === 'file_number' || kind === 'order_number';
}

/** Der Nachweis, den der Weg einer Pflicht üblicherweise liefert. */
export function defaultEvidence(channel: string): EvidenceKind {
  switch (channel) {
    case 'elster':
    case 'bzst':
    case 'lohnanbieter':
      return 'transfer_ticket';
    case 'unternehmensregister':
    case 'transparenzregister':
      return 'order_number';
    case 'notar':
    case 'gewerbeamt':
      return 'file_number';
    default:
      return 'confirmation';
  }
}

/** Der heutige Tag in der Zeitzone des Rechners („JJJJ-MM-TT“). */
export function todayISO(now = new Date()): string {
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${now.getFullYear()}-${month}-${day}`;
}

/** Kann der Mensch die Pflicht in der App bestätigen oder als entfallen kennzeichnen? */
export function isConfirmable(o: Obligation): boolean {
  return o.evidenceSource !== 'data' && o.status !== 'done' && o.status !== 'waived';
}
