import React, { useEffect, useState } from 'react';
import { Api } from '../../services/api';
import { useWriteLock } from '../../components/WriteLock';
import { findingRoute, targetLabel } from '../../utils/findings';
import { channelNote, formatDate } from '../../utils/formatters';
import type { NavigateFn } from '../../navigation';
import type {
  CheckFinding,
  CheckRun,
  Obligation,
  EvidenceKind,
} from '../../types';
import {
  Button,
  Dialog,
  Field,
  Help,
  Input,
  Notice,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Textarea,
  cn,
  toast,
} from '../../components/ui';

export function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Die Ladefehler eines Reiters und des Pflichtenkalenders in einer Meldung.
 * Scheitern beide am selben Grund, steht er einmal da.
 */
export function loadErrorText(...errors: (string | undefined)[]): string {
  const distinct = [...new Set(errors.filter((e): e is string => Boolean(e)))];
  return distinct.length > 0 ? `Konnte nicht vollständig geladen werden. ${distinct.join(' ')}` : '';
}

/** Heute als JJJJ-MM-TT in der Zeitzone des Rechners. */
export function todayISO(): string {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

const MONTH_NAMES = [
  'Januar',
  'Februar',
  'März',
  'April',
  'Mai',
  'Juni',
  'Juli',
  'August',
  'September',
  'Oktober',
  'November',
  'Dezember',
];

/** „2026-07" → „Juli 2026". */
export function monthLabel(month: string): string {
  const [year, index] = month.split('-').map(Number);
  return `${MONTH_NAMES[(index || 1) - 1]} ${year}`;
}

/** „2026-07" → „Jul". */
export function monthShort(month: string): string {
  const index = Number(month.slice(5, 7));
  return MONTH_NAMES[(index || 1) - 1].slice(0, 3);
}

/** Der letzte Tag eines Monats „JJJJ-MM". */
export function monthEnd(month: string): string {
  const [year, index] = month.split('-').map(Number);
  const last = new Date(Date.UTC(year, index, 0)).getUTCDate();
  return `${month}-${String(last).padStart(2, '0')}`;
}

/** Die Monate eines Zeitraums von–bis als „JJJJ-MM". */
export function monthsBetween(from: string, to: string): string[] {
  const out: string[] = [];
  let [year, month] = from.slice(0, 7).split('-').map(Number);
  const end = to.slice(0, 7);
  for (let guard = 0; guard < 24; guard++) {
    const key = `${year}-${String(month).padStart(2, '0')}`;
    out.push(key);
    if (key >= end) break;
    month += 1;
    if (month > 12) {
      month = 1;
      year += 1;
    }
  }
  return out;
}

/**
 * Der Reiter, unter dem eine Pflicht steht. Spiegelt `obligationTarget` in
 * internal/service/task_service.go, damit ein Verweis aus der Übersicht
 * dort landet, wo die Pflicht erledigt wird.
 */
export function obligationTab(kind: string): 'month' | 'year' | 'declarations' | 'company' {
  switch (kind) {
    case 'kst-erklaerung':
    case 'gewst-erklaerung':
    case 'ust-erklaerung':
    case 'e-bilanz':
    case 'kst-vorauszahlung':
    case 'gewst-vorauszahlung':
      return 'declarations';
    case 'abschlussbuchungen':
    case 'aufstellung':
    case 'feststellung':
    case 'ergebnisverwendung':
    case 'offenlegung':
    case 'kapitalertragsteuer':
      return 'year';
    case 'transparenzregister':
    case 'gesellschafterliste':
    case 'steuerliche-erfassung':
    case 'gewerbeanmeldung':
      return 'company';
    default:
      return 'month';
  }
}

/** Die Pflichtart aus dem Schlüssel „ustva.2026-Q3". */
export function obligationKindOf(key: string): string {
  return key.split('.')[0] ?? '';
}

/** Lässt sich die Pflicht in der App bestätigen oder als entfallen kennzeichnen? */
export function isManual(o: Obligation): boolean {
  return o.evidenceSource !== 'data' && o.status !== 'done' && o.status !== 'waived';
}

/** Tage bis zum Tag; negativ heißt vorbei. */
function daysUntil(iso: string): number {
  const today = new Date(`${todayISO()}T00:00:00`);
  const due = new Date(`${iso}T00:00:00`);
  return Math.round((due.getTime() - today.getTime()) / 86_400_000);
}

/** Die Zeile unter einer Pflicht: Fälligkeit, Nachweis oder Grund. */
export function obligationCaption(o: Obligation): string {
  switch (o.status) {
    case 'done': {
      const proof = o.proof;
      if (!proof) return 'erledigt';
      const ref = proof.reference ? ` · ${proof.reference}` : '';
      return `erledigt am ${formatDate(proof.date)}${ref}`;
    }
    case 'waived':
      return o.waivedReason ? `entfällt: ${o.waivedReason}` : 'entfällt';
    case 'overdue': {
      const days = o.dueDate ? -daysUntil(o.dueDate) : 0;
      return days > 0 ? `seit ${days} ${days === 1 ? 'Tag' : 'Tagen'} fällig` : 'überfällig';
    }
    default: {
      if (o.immediate && !o.dueDate) return 'unverzüglich';
      const prepared = o.preparation
        ? ` · vorbereitet ${channelNote(o.preparation.channel) ?? ''}`.trimEnd()
        : '';
      return `fällig am ${formatDate(o.dueDate)}${prepared}`;
    }
  }
}

/** Abzeichen und Zeile einer Pflicht, nebeneinander. */
export const ObligationState: React.FC<{ obligation: Obligation; className?: string }> = ({
  obligation,
  className,
}) => (
  <span className={cn('inline-flex items-center gap-2', className)}>
    {obligation.status === 'done' ? (
      <StatusBadge status="erledigt" />
    ) : obligation.status === 'overdue' ? (
      <StatusBadge status="ueberfaellig" />
    ) : obligation.status === 'waived' ? null : (
      <StatusBadge status="offen" />
    )}
    <span className="text-caption text-ink-subtle num">{obligationCaption(obligation)}</span>
  </span>
);

const EVIDENCE_LABELS: Record<EvidenceKind, string> = {
  transfer_ticket: 'Transferticket',
  file_number: 'Aktenzeichen',
  order_number: 'Auftragsnummer',
  booking: 'Buchung',
  commitment: 'Festschreibung',
  resolution: 'Beschluss',
  status: 'Abschlussstand',
  confirmation: 'Bestätigung ohne Nummer',
};

/** Die Nachweise, die ein Mensch von Hand angibt. */
const MANUAL_EVIDENCE: EvidenceKind[] = [
  'transfer_ticket',
  'file_number',
  'order_number',
  'resolution',
  'confirmation',
];

/** Welcher Nachweis zu einer Pflicht passt, wenn der Anwender nichts wählt. */
function defaultEvidence(kind: string): EvidenceKind {
  switch (kind) {
    case 'e-bilanz':
    case 'lohnsteuer-anmeldung':
    case 'kapitalertragsteuer':
    case 'sondervorauszahlung':
    case 'kst-erklaerung':
    case 'gewst-erklaerung':
    case 'ust-erklaerung':
      return 'transfer_ticket';
    default:
      return 'confirmation';
  }
}

/**
 * Bestätigen oder Entfallen einer Pflicht, die Buchfink nicht aus den Daten
 * sieht. Beides ist eine Erklärung des Menschen und steht mit Nachweis im
 * Protokoll.
 */
export const ObligationDecisionDialog: React.FC<{
  obligation: Obligation | null;
  onClose: () => void;
  onDone: (next: Obligation) => void | Promise<void>;
}> = ({ obligation, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [mode, setMode] = useState<'confirm' | 'waive'>('confirm');
  const [date, setDate] = useState(todayISO());
  const [kind, setKind] = useState<EvidenceKind>('confirmation');
  const [reference, setReference] = useState('');
  const [reason, setReason] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!obligation) return;
    setMode('confirm');
    setDate(todayISO());
    setKind(defaultEvidence(obligation.kind));
    setReference('');
    setReason('');
    setFieldError('');
    setError('');
  }, [obligation]);

  const needsReference = kind === 'transfer_ticket' || kind === 'file_number' || kind === 'order_number';

  async function submit() {
    if (!obligation) return;
    if (mode === 'confirm' && needsReference && !reference.trim()) {
      setFieldError(`Die Nummer fehlt. Sie steht im ${EVIDENCE_LABELS[kind]}-Feld des Nachweises.`);
      return;
    }
    if (mode === 'waive' && !reason.trim()) {
      setFieldError('Der Grund fehlt. Er steht danach im Protokoll.');
      return;
    }
    setFieldError('');
    setBusy(true);
    try {
      const next =
        mode === 'confirm'
          ? await Api.confirmObligation(obligation.key, {
              date,
              kind,
              reference: reference.trim(),
            })
          : await Api.waiveObligation(obligation.key, reason.trim());
      toast.success(
        mode === 'confirm'
          ? `${obligation.title} als erledigt vermerkt.`
          : `${obligation.title} entfällt.`,
      );
      await onDone(next);
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={obligation !== null}
      onOpenChange={(open) => !open && onClose()}
      title={obligation?.title ?? ''}
      width="max-w-lg"
      footer={
        <>
          <Button
            variant="quiet"
            className="mr-auto"
            onClick={() => {
              setMode(mode === 'confirm' ? 'waive' : 'confirm');
              setFieldError('');
            }}
          >
            {mode === 'confirm' ? 'Entfällt stattdessen' : 'Doch bestätigen'}
          </Button>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void submit()}
          >
            {mode === 'confirm' ? 'Als erledigt vermerken' : 'Als entfallen vermerken'}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <p className="text-body text-ink-muted">
          {mode === 'confirm'
            ? 'Der Nachweis steht danach an der Pflicht und im Protokoll.'
            : 'Die Pflicht verschwindet aus den offenen Aufgaben. Der Grund steht im Protokoll.'}
        </p>
        {mode === 'confirm' ? (
          <>
            <div className="grid grid-cols-2 gap-4">
              <Field label="Datum">
                <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
              </Field>
              <Field label="Nachweis">
                <Select
                  items={MANUAL_EVIDENCE.map((value) => ({ value, label: EVIDENCE_LABELS[value] }))}
                  value={kind}
                  onValueChange={(next) => setKind(next as EvidenceKind)}
                />
              </Field>
            </div>
            {kind !== 'confirmation' && (
              <Field label={EVIDENCE_LABELS[kind]} error={fieldError || undefined} optional={!needsReference}>
                <Input
                  className="code-num"
                  value={reference}
                  onChange={(e) => {
                    setReference(e.target.value);
                    setFieldError('');
                  }}
                />
              </Field>
            )}
          </>
        ) : (
          <Field label="Grund" error={fieldError || undefined}>
            <Textarea
              rows={3}
              value={reason}
              onChange={(e) => {
                setReason(e.target.value);
                setFieldError('');
              }}
            />
          </Field>
        )}
        {error && <Notice tone="negative" text={error} />}
      </div>
    </Dialog>
  );
};

/** Ein Befund des Prüflaufs als Zeile, mit dem Weg zu seiner Stelle. */
export const FindingRows: React.FC<{
  findings: CheckFinding[];
  onNavigate?: NavigateFn;
  onLeave?: () => void;
}> = ({ findings, onNavigate, onLeave }) => (
  <ul className="flex flex-col">
    {findings.map((finding) => {
      const route = findingRoute(finding);
      return (
        <li
          key={`${finding.rule}-${finding.objectId}-${finding.message}`}
          className="flex items-start gap-3 py-2 border-t border-line first:border-t-0"
        >
          <span
            className={cn(
              'w-24 shrink-0 text-caption',
              finding.severity === 'blocking' ? 'text-negative-text' : 'text-attention-text',
            )}
          >
            {finding.severity === 'blocking' ? 'Blockierend' : 'Hinweis'}
          </span>
          <span className="flex-1 text-body text-ink-muted">{finding.message}</span>
          {onNavigate && route && (
            <Button
              variant="quiet"
              size="sm"
              className="shrink-0 -my-1"
              onClick={() => {
                onLeave?.();
                onNavigate(route);
              }}
            >
              {targetLabel(route)}
            </Button>
          )}
        </li>
      );
    })}
  </ul>
);

export interface CommitTarget {
  type: 'month' | 'year';
  label: string;
  cutoff: string;
}

const REASON_FIELD = 'commit-override-reason';

/**
 * Die Festschreibung mit dem Prüfbericht davor. Der Bericht sagt, was danach
 * nicht mehr zu ändern wäre; ein blockierender Befund lässt sich nur mit
 * Begründung übergehen.
 */
export const CommitDialog: React.FC<{
  target: CommitTarget | null;
  /** Ein Satz über das hinaus, was jede Festschreibung bedeutet. */
  note?: string;
  onClose: () => void;
  onCommitted: () => void | Promise<void>;
  onNavigate?: NavigateFn;
}> = ({ target, note, onClose, onCommitted, onNavigate }) => {
  const writeLock = useWriteLock();
  const [run, setRun] = useState<CheckRun | null>(null);
  const [loading, setLoading] = useState(false);
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!target) return;
    let cancelled = false;
    setRun(null);
    setReason('');
    setReasonError('');
    setError('');
    setLoading(true);
    Api.runChecks(target.cutoff, target.type)
      .then((next) => !cancelled && setRun(next))
      .catch((e) => !cancelled && setError(message(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [target]);

  const findings = run?.findings ?? [];
  const blocking = findings.filter((f) => f.severity === 'blocking');
  const warnings = findings.filter((f) => f.severity !== 'blocking');

  async function commit() {
    if (!target) return;
    if (blocking.length > 0 && !reason.trim()) {
      setReasonError('Blockierende Befunde lassen sich nur mit einer Begründung übergehen.');
      document.getElementById(REASON_FIELD)?.focus();
      return;
    }
    setBusy(true);
    try {
      const record = await Api.commitPeriod(target.type, target.label, target.cutoff, reason.trim());
      toast.success(
        record?.timestampStatus === 'confirmed'
          ? `${target.label} festgeschrieben, beglaubigt durch ${record.tsaName}.`
          : `${target.label} festgeschrieben. Der Zeitstempel folgt, sobald wieder Netz da ist.`,
      );
      await onCommitted();
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={target !== null}
      onOpenChange={(open) => !open && onClose()}
      title={target ? `${target.label} festschreiben` : ''}
      width="max-w-2xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={loading || !run || writeLock.locked}
            title={writeLock.hint}
            onClick={() => void commit()}
          >
            Festschreiben
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-5">
        <p className="text-body text-ink">
          {`Alle Buchungen bis ${target ? formatDate(target.cutoff) : ''} lassen sich danach nur noch per Storno korrigieren.`}
          {note ? ` ${note}` : ''}
          <Help summary="Der festgeschriebene Stand bleibt unverändert. Korrekturen laufen danach über einen Storno." label="Erklärung zur Festschreibung">
            Buchungen dürfen nicht so verändert werden, dass der ursprüngliche Inhalt nicht mehr
            feststellbar ist (§ 146 Abs. 4 AO, GoBD Rz. 107). Die Festschreibung macht das
            sichtbar: Buchfink verankert den Stand mit einem Zeitstempel. Übertragen wird dabei nur
            eine Prüfsumme, keine Buchung.
          </Help>
        </p>
        {loading ? (
          <SkeletonRows rows={3} />
        ) : run ? (
          <>
            <StatRow>
              <Stat label="Geprüfte Buchungen" value={String(run.checkedEntries)} />
              <Stat label="Belege" value={String(run.checkedReceipts)} />
              <Stat label="Bankumsätze" value={String(run.checkedBankTx)} />
              <Stat
                label="Befunde"
                value={`${blocking.length} · ${warnings.length}`}
                context="blockierend · Hinweise"
                tone={blocking.length > 0 ? 'negative' : 'neutral'}
              />
            </StatRow>
            {findings.length === 0 ? (
              <p className="text-body text-ink-muted">Der Prüfbericht ist ohne Befund.</p>
            ) : (
              <FindingRows findings={[...blocking, ...warnings]} onNavigate={onNavigate} onLeave={onClose} />
            )}
            {blocking.length > 0 && (
              <Field
                label="Begründung für das Übergehen"
                hint="steht am Prüflauf und im Protokoll"
                error={reasonError || undefined}
              >
                <Textarea
                  id={REASON_FIELD}
                  rows={3}
                  value={reason}
                  onChange={(e) => {
                    setReason(e.target.value);
                    setReasonError('');
                  }}
                />
              </Field>
            )}
          </>
        ) : null}
        {error && <Notice tone="negative" text={error} />}
      </div>
    </Dialog>
  );
};

/**
 * Das Transferticket einer Meldung aus Mein ELSTER oder dem BZSt-Portal:
 * Datum und Nummer, darunter der Knopf. Die Nummer ist der Nachweis, dass die
 * Meldung angekommen ist.
 */
export const TicketForm: React.FC<{
  label: string;
  hint: string;
  busy: boolean;
  onSubmit: (date: string, ticket: string) => void | Promise<void>;
}> = ({ label, hint, busy, onSubmit }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [ticket, setTicket] = useState('');
  const [error, setError] = useState('');
  return (
    <div className="flex flex-wrap items-end gap-4">
      <Field label="Übermittelt am" className="w-44">
        <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
      </Field>
      <Field label="Transferticket" hint={hint} error={error || undefined} className="w-64">
        <Input
          className="code-num"
          value={ticket}
          onChange={(e) => {
            setTicket(e.target.value);
            setError('');
          }}
        />
      </Field>
      <Button
        variant="secondary"
        loading={busy}
        disabled={writeLock.locked}
        title={writeLock.hint}
        onClick={() => {
          if (!ticket.trim()) {
            setError('Ohne Transferticket gibt es keine Bestätigung.');
            return;
          }
          void onSubmit(date, ticket.trim());
        }}
      >
        {label}
      </Button>
    </div>
  );
};
