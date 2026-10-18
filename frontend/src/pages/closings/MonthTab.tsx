import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ChevronRight, Lock } from 'lucide-react';
import { Api } from '../../services/api';
import { useWriteLock } from '../../components/WriteLock';
import { downloadCSV } from '../../utils/download';
import { formatCents, formatDate } from '../../utils/formatters';
import type { NavigateFn } from '../../navigation';
import type {
  MonthCloseState,
  MonthCloseStep,
  Obligation,
  VatReturn,
  VatReturnLine,
  ZMPeriodStatus,
  ZMReturn,
} from '../../types';
import {
  Button,
  EmptyState,
  Help,
  Notice,
  PageHeader,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../../components/ui';
import {
  CommitDialog,
  FindingRows,
  ObligationDecisionDialog,
  ObligationState,
  TicketForm,
  isManual,
  message,
  monthEnd,
  monthLabel,
  monthShort,
  monthsBetween,
  todayISO,
  type CommitTarget,
  loadErrorText,
} from './shared';

/**
 * Der Monatsabschluss als Ablauf: Prüfbericht, Festschreiben, Voranmeldung und
 * Zusammenfassende Meldung, Transferticket. Gemeldet wird erst, was
 * festgeschrieben ist; sonst weicht die Buchführung später von der Meldung ab.
 *
 * Die Kennzahlen sind eine Eingabehilfe für Mein ELSTER. Buchfink übermittelt
 * nicht selbst; bestätigt wird mit dem Transferticket aus dem
 * Übermittlungsprotokoll.
 */

/** Kennzahl 83 ist die Zahllast; sie steht im Blatt auch bei null. */
const PAYABLE_CODE = '83';

/** Monatspflichten, die nicht schon als eigener Schritt stehen. */
const SIDE_KINDS = ['lohnsteuer-anmeldung', 'sondervorauszahlung'];

type MonthMark = 'committed' | 'overdue' | 'open' | 'running' | 'unknown';

const MARK_CLASS: Record<MonthMark, string> = {
  committed: 'bg-positive',
  overdue: 'bg-negative',
  open: 'bg-attention',
  running: 'bg-ink-faint',
  unknown: 'bg-line-strong',
};

const MARK_LABEL: Record<MonthMark, string> = {
  committed: 'festgeschrieben',
  overdue: 'Festschreibung überfällig',
  open: 'Festschreibung offen',
  running: 'läuft noch',
  unknown: 'Stand nicht geladen',
};

/** Der Monat davor als „JJJJ-MM". */
function previousMonthOf(date: string): string {
  const [year, month] = date.slice(0, 7).split('-').map(Number);
  return month === 1 ? `${year - 1}-12` : `${year}-${String(month - 1).padStart(2, '0')}`;
}

/** Zu welchem Monatsabschluss eine Pflicht gehört. */
function monthOfObligation(o: Obligation): string {
  if (o.kind === 'sondervorauszahlung' && o.dueDate) return previousMonthOf(o.dueDate);
  return (o.periodTo || o.dueDate || '').slice(0, 7);
}

/**
 * Der Monat, den ein Ziel meint: „2026-07" unmittelbar oder der Schlüssel einer
 * Pflicht wie „ustva.2026-Q3" aus der Übersicht.
 */
function monthOfTarget(id: string | undefined, obligations: Obligation[]): string | null {
  if (!id) return null;
  if (/^\d{4}-\d{2}$/.test(id)) return id;
  const known = obligations.find((o) => o.key === id);
  if (known) return monthOfObligation(known);
  const period = id.split('.')[1] ?? '';
  const quarter = /^(\d{4})-Q([1-4])$/.exec(period);
  if (quarter) return `${quarter[1]}-${String(Number(quarter[2]) * 3).padStart(2, '0')}`;
  if (/^\d{4}-\d{2}$/.test(period)) return period;
  return null;
}

export interface MonthTabProps {
  year: number;
  obligations: Obligation[];
  /** Der Pflichtenkalender ließ sich nicht laden. */
  obligationsError?: string;
  onObligationsChanged: () => Promise<void>;
  id?: string;
  onNavigate: NavigateFn;
}

export const MonthTab: React.FC<MonthTabProps> = ({
  year,
  obligations,
  obligationsError,
  onObligationsChanged,
  id,
  onNavigate,
}) => {
  const writeLock = useWriteLock();
  const [span, setSpan] = useState<{ from: string; to: string }>({
    from: `${year}-01-01`,
    to: `${year}-12-31`,
  });
  const [state, setState] = useState<MonthCloseState | null>(null);
  const [stateError, setStateError] = useState('');
  const [loading, setLoading] = useState(true);
  const [vatReturn, setVatReturn] = useState<VatReturn | null>(null);
  const [vatError, setVatError] = useState('');
  const [zmPeriods, setZmPeriods] = useState<ZMPeriodStatus[]>([]);
  const [zmReturn, setZmReturn] = useState<ZMReturn | null>(null);
  const [commitTarget, setCommitTarget] = useState<CommitTarget | null>(null);
  const [deciding, setDeciding] = useState<Obligation | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  useEffect(() => {
    Api.getFiscalYears()
      .then((years) => {
        const fy = years.find((entry) => entry.year === year);
        if (fy) setSpan({ from: fy.startDate, to: fy.endDate });
      })
      .catch(() => undefined);
    Api.getZMPeriods(year)
      .then(setZmPeriods)
      .catch(() => setZmPeriods([]));
  }, [year]);

  const today = todayISO();
  const months = useMemo(() => monthsBetween(span.from, span.to), [span]);
  const commits = useMemo(() => {
    const out = new Map<string, Obligation>();
    for (const o of obligations) if (o.kind === 'festschreibung') out.set(o.periodKey, o);
    return out;
  }, [obligations]);

  const markOf = useCallback(
    (month: string): MonthMark => {
      const o = commits.get(month);
      if (o?.status === 'done') return 'committed';
      if (o?.status === 'overdue') return 'overdue';
      if (monthEnd(month) >= today) return 'running';
      // Ohne Pflichtenkalender ist unbekannt, ob der Monat festgeschrieben ist.
      return obligationsError ? 'unknown' : 'open';
    },
    [commits, today, obligationsError],
  );

  // Ohne Ziel zeigt der Reiter den ältesten Monat, der noch nicht
  // festgeschrieben ist; sonst den letzten beendeten.
  const fallback = useMemo(() => {
    const ended = months.filter((m) => monthEnd(m) < today);
    return (
      ended.find((m) => markOf(m) === 'overdue' || markOf(m) === 'open') ??
      ended[ended.length - 1] ??
      months[0]
    );
  }, [months, markOf, today]);
  const month = monthOfTarget(id, obligations) ?? fallback;

  const load = useCallback(async () => {
    setLoading(true);
    setStateError('');
    try {
      const next = await Api.getMonthCloseState(month);
      setState(next);
      if (next.vatApplies && next.vatPeriodKey) {
        try {
          setVatReturn(await Api.getVatReturn(next.vatPeriodKey));
          setVatError('');
        } catch (e) {
          setVatReturn(null);
          setVatError(message(e));
        }
      } else {
        setVatReturn(null);
        setVatError('');
      }
    } catch (e) {
      setState(null);
      setVatReturn(null);
      setStateError(message(e));
    } finally {
      setLoading(false);
    }
  }, [month]);

  useEffect(() => {
    void load();
  }, [load]);

  const zmPeriod = state ? zmPeriods.find((p) => p.to === state.to) ?? null : null;
  useEffect(() => {
    if (!zmPeriod) {
      setZmReturn(null);
      return;
    }
    Api.getZMReturn(zmPeriod.key)
      .then(setZmReturn)
      .catch(() => setZmReturn(null));
  }, [zmPeriod?.key, zmPeriod?.status]);

  const reloadAll = async () => {
    await load();
    await onObligationsChanged();
    setZmPeriods(await Api.getZMPeriods(year).catch(() => zmPeriods));
  };

  const ended = state ? state.to < today : false;
  const committed = state?.committed ?? false;
  const sideObligations = obligations.filter(
    (o) => SIDE_KINDS.includes(o.kind) && monthOfObligation(o) === month,
  );
  const showZm =
    zmPeriod !== null &&
    zmReturn !== null &&
    (zmReturn.lines.length > 0 || zmPeriod.status === 'submitted');

  const commitObligation = commits.get(month);
  const contextParts: string[] = [];
  if (committed) contextParts.push(`festgeschrieben bis ${formatDate(state?.committedTo ?? '')}`);
  else if (!ended && state) contextParts.push(`läuft bis ${formatDate(state.to)}`);
  else if (commitObligation) contextParts.push(`Festschreibung fällig am ${formatDate(commitObligation.dueDate)}`);
  if (state?.vatApplies && state.vatDueDate && state.vatStatus !== 'submitted') {
    contextParts.push(`Voranmeldung fällig am ${formatDate(state.vatDueDate)}`);
  }

  async function confirmVat(date: string, ticket: string) {
    if (!state?.vatPeriodKey) return;
    setBusy('vat');
    try {
      const saved = await Api.saveVatReturn(state.vatPeriodKey);
      await Api.confirmVatReturnSubmitted(saved.id, date, ticket);
      toast.success(`Voranmeldung ${state.vatPeriodLabel ?? ''} als übermittelt vermerkt.`);
      await reloadAll();
    } catch (e) {
      setVatError(message(e));
    } finally {
      setBusy(null);
    }
  }

  async function exportVat() {
    if (!state?.vatPeriodKey) return;
    setBusy('export');
    try {
      const target =
        state.vatStatus === 'submitted' && state.vatReturnId
          ? state.vatReturnId
          : (await Api.saveVatReturn(state.vatPeriodKey)).id;
      downloadCSV(`ustva-${state.vatPeriodKey}.csv`, await Api.exportVatReturnCSV(target));
    } catch (e) {
      setVatError(message(e));
    } finally {
      setBusy(null);
    }
  }

  async function correction(periodKey: string) {
    setBusy('correction');
    try {
      const created = await Api.createVatCorrection(periodKey);
      toast.success('Berichtigte Voranmeldung angelegt.');
      onNavigate({ area: 'closings', tab: 'month', id: created.periodTo.slice(0, 7) });
    } catch (e) {
      setVatError(message(e));
    } finally {
      setBusy(null);
    }
  }

  async function confirmZm(date: string, ticket: string) {
    if (!zmPeriod) return;
    setBusy('zm');
    try {
      const saved = await Api.saveZMReturn(zmPeriod.key);
      await Api.confirmZMSubmitted(saved.id, date, ticket);
      toast.success(`Zusammenfassende Meldung ${zmPeriod.label} als übermittelt vermerkt.`);
      await reloadAll();
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  const steps = state?.steps ?? [];
  const stepOf = (key: string) => steps.find((step) => step.key === key);
  const blocking = state?.blocking ?? 0;

  return (
    <div>
      <PageHeader
        title={monthLabel(month)}
        context={contextParts.join(' · ') || undefined}
        helpSummary="Prüfen, festschreiben, melden: in dieser Reihenfolge wird ein Monat abgeschlossen."
        explain={
          <>
            Gemeldet wird ein Stand, der sich nicht mehr ändert. Deshalb steht die Festschreibung
            vor der Voranmeldung (§ 146 Abs. 4 AO, GoBD Rz. 107). Die Kennzahlen tragen Sie oder
            Claude in Mein ELSTER ein. Das Transferticket aus dem Übermittlungsprotokoll ist danach
            der Nachweis, dass die Meldung angekommen ist.
          </>
        }
        action={
          state && !committed ? (
            <Button
              variant="primary"
              icon={<Lock className="w-4 h-4" strokeWidth={1.5} />}
              disabled={!ended || writeLock.locked}
              title={
                writeLock.hint ??
                (!ended ? `Der Monat läuft noch bis ${formatDate(state.to)}.` : undefined)
              }
              onClick={() =>
                setCommitTarget({ type: 'month', label: state.label, cutoff: state.to })
              }
            >
              {`${state.label.split(' ')[0]} festschreiben`}
            </Button>
          ) : undefined
        }
      />

      <MonthStrip
        months={months}
        selected={month}
        markOf={markOf}
        onSelect={(next) => onNavigate({ area: 'closings', tab: 'month', id: next })}
      />

      {(stateError || obligationsError) && (
        <Notice tone="negative" className="mt-6" text={loadErrorText(stateError, obligationsError)} />
      )}

      {loading && !state ? (
        <div className="mt-8">
          <SkeletonRows rows={6} />
        </div>
      ) : state ? (
        <ol key={state.month} className="mt-8 flex flex-col">
          <FlowStep
            number={1}
            step={stepOf('check')}
            title="Prüfbericht"
            summary={
              state.findings.length === 0
                ? 'Ohne Befund.'
                : blocking > 0
                  ? `${blocking} blockierend, ${state.findings.length - blocking} Hinweise`
                  : `${state.findings.length} ${state.findings.length === 1 ? 'Hinweis' : 'Hinweise'}, nichts blockiert`
            }
          >
            {state.findings.length > 0 && (
              <FindingRows findings={state.findings} onNavigate={onNavigate} />
            )}
          </FlowStep>

          <FlowStep
            number={2}
            step={stepOf('commit')}
            title="Festschreiben"
            summary={
              committed
                ? `Festgeschrieben bis ${formatDate(state.committedTo ?? state.to)}`
                : state.committedTo && state.committedTo < previousDay(state.from)
                  ? `Festgeschrieben bis ${formatDate(state.committedTo)}. Die Monate dazwischen werden mitgeschrieben.`
                  : stepOf('commit')?.note ?? ''
            }
          />

          {state.vatApplies ? (
            <FlowStep
              number={3}
              step={stepOf('vat')}
              title={`Umsatzsteuer-Voranmeldung ${state.vatPeriodLabel ?? ''}`}
              summary={
                state.vatStatus === 'submitted'
                  ? `Übermittelt am ${formatDate(state.vatSubmittedAt ?? '')}${vatReturn?.transferTicket ? ` · Transferticket ${vatReturn.transferTicket}` : ''}`
                  : `Fällig am ${formatDate(state.vatDueDate ?? '')}`
              }
              open
            >
              <VatSheet
                vatReturn={vatReturn}
                onNavigate={onNavigate}
                onExport={() => void exportVat()}
                exporting={busy === 'export'}
              />
              {vatError && <Notice tone="negative" className="mt-4" text={vatError} />}
              {vatReturn && vatReturn.lateEntries.length > 0 && (
                <Notice
                  className="mt-4"
                  text={`${vatReturn.lateEntries.length} Buchungen gehören in einen schon übermittelten Zeitraum (${vatReturn.lateEntries[0].periodKey}).`}
                  action={
                    <Button
                      variant="secondary"
                      size="sm"
                      loading={busy === 'correction'}
                      disabled={writeLock.locked}
                      title={writeLock.hint}
                      onClick={() => void correction(vatReturn.lateEntries[0].periodKey)}
                    >
                      Berichtigung anlegen
                    </Button>
                  }
                />
              )}
              <div className="mt-5">
                {state.vatStatus === 'submitted' ? null : !committed ? (
                  <p className="text-body text-ink-muted">
                    Die Übermittlung bestätigen Sie nach der Festschreibung.
                  </p>
                ) : (
                  <TicketForm
                    label="Übermittlung bestätigen"
                    hint="aus Mein ELSTER"
                    busy={busy === 'vat'}
                    onSubmit={confirmVat}
                  />
                )}
              </div>
            </FlowStep>
          ) : (
            <FlowStep number={3} step={stepOf('vat')} title="Umsatzsteuer-Voranmeldung" summary={stepOf('vat')?.note ?? ''} />
          )}

          {showZm && zmPeriod && zmReturn && (
            <FlowStep
              number={4}
              title={`Zusammenfassende Meldung ${zmPeriod.label}`}
              state={
                zmPeriod.status === 'submitted'
                  ? 'done'
                  : zmPeriod.committed
                    ? 'open'
                    : 'blocked'
              }
              summary={
                zmPeriod.status === 'submitted'
                  ? `Übermittelt am ${formatDate(zmPeriod.submittedAt ?? '')}`
                  : `Fällig am ${formatDate(zmPeriod.dueDate)}`
              }
              open
            >
              <ZmSheet zmReturn={zmReturn} />
              {zmPeriod.status !== 'submitted' && zmPeriod.committed && (
                <div className="mt-5">
                  <TicketForm
                    label="Meldung bestätigen"
                    hint="aus dem BZSt-Portal"
                    busy={busy === 'zm'}
                    onSubmit={confirmZm}
                  />
                </div>
              )}
            </FlowStep>
          )}
        </ol>
      ) : null}

      {sideObligations.length > 0 && (
        <section className="mt-8 pt-6 border-t border-line">
          <h3 className="text-label text-ink-muted mb-2">Weitere Pflichten zu diesem Monat</h3>
          <ul className="flex flex-col">
            {sideObligations.map((o) => (
              <li
                key={o.key}
                className="flex items-center gap-4 py-2 border-t border-line first:border-t-0"
              >
                <span className="flex-1 min-w-0 flex items-center gap-1.5 text-body text-ink">
                  <span className="truncate">{o.title}</span>
                  <Help summary="Was diese Pflicht verlangt und woran Buchfink erkennt, dass sie erledigt ist." label={`Erklärung zu ${o.title}`}>
                    {`${o.description} ${o.basis}`}
                  </Help>
                </span>
                <ObligationState obligation={o} />
                {isManual(o) ? (
                  <Button
                    variant="quiet"
                    size="sm"
                    disabled={writeLock.locked}
                    title={writeLock.hint}
                    onClick={() => setDeciding(o)}
                  >
                    Erledigt vermerken
                  </Button>
                ) : (
                  <span className="w-[8.5rem]" aria-hidden="true" />
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      <CommitDialog
        target={commitTarget}
        onClose={() => setCommitTarget(null)}
        onCommitted={reloadAll}
        onNavigate={onNavigate}
      />
      <ObligationDecisionDialog
        obligation={deciding}
        onClose={() => setDeciding(null)}
        onDone={onObligationsChanged}
      />
    </div>
  );
};

function previousDay(date: string): string {
  const d = new Date(`${date}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() - 1);
  return d.toISOString().slice(0, 10);
}

/** Die Monate des Geschäftsjahres als Leiste, je mit seinem Stand. */
const MonthStrip: React.FC<{
  months: string[];
  selected: string;
  markOf: (month: string) => MonthMark;
  onSelect: (month: string) => void;
}> = ({ months, selected, markOf, onSelect }) => (
  <div
    className="mt-6 grid gap-1.5"
    style={{ gridTemplateColumns: `repeat(${months.length}, minmax(0, 1fr))` }}
    role="list"
    aria-label="Monate des Geschäftsjahres"
  >
    {months.map((month) => {
      const mark = markOf(month);
      const active = month === selected;
      return (
        <button
          key={month}
          type="button"
          role="listitem"
          aria-current={active ? 'true' : undefined}
          title={`${monthLabel(month)}: ${MARK_LABEL[mark]}`}
          onClick={() => onSelect(month)}
          className={cn(
            'flex flex-col items-center gap-1.5 rounded-control border px-1 py-2',
            'transition-colors duration-120 ease-quiet',
            'focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent',
            active ? 'border-ink bg-surface' : 'border-line hover:bg-sunken',
          )}
        >
          <span className={cn('text-caption', active ? 'text-ink font-medium' : 'text-ink-muted')}>
            {monthShort(month)}
          </span>
          <span className={cn('mark-diamond', MARK_CLASS[mark])} aria-hidden="true" />
        </button>
      );
    })}
  </div>
);

type FlowState = MonthCloseStep['state'];

const FLOW_MARK: Record<FlowState, { mark: string; label: string }> = {
  done: { mark: 'bg-positive', label: 'Erledigt' },
  blocked: { mark: 'bg-negative', label: 'Wartet' },
  open: { mark: 'bg-attention', label: 'Offen' },
  not_applicable: { mark: 'bg-ink-faint', label: 'Entfällt' },
};

/**
 * Ein Schritt des Ablaufs: Nummer, Titel, eine Zeile Stand. Einzelheiten
 * stehen darunter und gehen auf Klick auf, außer der Schritt ist der, an dem
 * gearbeitet wird (`open`).
 */
const FlowStep: React.FC<{
  number: number;
  title: string;
  summary: string;
  step?: MonthCloseStep;
  state?: FlowState;
  open?: boolean;
  children?: React.ReactNode;
}> = ({ number, title, summary, step, state, open = false, children }) => {
  const flow = state ?? step?.state ?? 'open';
  const [expanded, setExpanded] = useState(open);
  const hasDetails = React.Children.toArray(children).some(Boolean);
  const mark = FLOW_MARK[flow];
  return (
    <li className="flex items-start gap-4 py-4 border-t border-line first:border-t-0 first:pt-0">
      <span className="w-6 shrink-0 text-caption text-ink-subtle num pt-0.5">{number}</span>
      <div className="flex-1 min-w-0">
        <div className="flex items-start gap-3">
          <button
            type="button"
            disabled={!hasDetails}
            aria-expanded={hasDetails ? expanded : undefined}
            onClick={() => setExpanded((value) => !value)}
            className={cn(
              'flex-1 min-w-0 flex items-start gap-1.5 text-left rounded-control',
              hasDetails ? 'cursor-pointer' : 'cursor-default',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent',
            )}
          >
            <ChevronRight
              className={cn(
                'w-3.5 h-3.5 mt-[3px] shrink-0 text-ink-faint transition-transform duration-180 ease-quiet',
                expanded && 'rotate-90',
                !hasDetails && 'invisible',
              )}
              strokeWidth={1.5}
              aria-hidden="true"
            />
            <span className="min-w-0">
              <span className="block text-body font-medium text-ink">{title}</span>
              <span className="block text-caption text-ink-muted mt-0.5">{summary}</span>
            </span>
          </button>
          <span className="shrink-0 flex items-center gap-2 text-caption text-ink-muted">
            <span className={cn('mark-diamond', mark.mark)} aria-hidden="true" />
            {mark.label}
          </span>
        </div>
        {hasDetails && expanded && <div className="mt-4 pl-5">{children}</div>}
      </div>
    </li>
  );
};

/**
 * Das Kennzahlenblatt der Voranmeldung, wie es in Mein ELSTER einzutragen ist.
 * Hinter jeder Kennzahl stehen die Buchungen, aus denen sie entsteht.
 */
const VatSheet: React.FC<{
  vatReturn: VatReturn | null;
  onNavigate: NavigateFn;
  onExport: () => void;
  exporting: boolean;
}> = ({ vatReturn, onNavigate, onExport, exporting }) => {
  const [complete, setComplete] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [numbers, setNumbers] = useState<Map<number, string> | null>(null);

  useEffect(() => {
    if (expanded === null || numbers !== null) return;
    Api.getAllJournalEntries()
      .then((entries) => setNumbers(new Map(entries.map((entry) => [entry.id, entry.entryNumber]))))
      .catch(() => setNumbers(new Map()));
  }, [expanded, numbers]);

  if (!vatReturn) {
    return (
      <EmptyState
        title="Kein Kennzahlenblatt"
        description="Für diesen Zeitraum ließ sich die Voranmeldung nicht rechnen."
      />
    );
  }

  const lines = vatReturn.figures.filter(
    (line) => complete || line.base !== 0 || line.tax !== 0 || line.code === PAYABLE_CODE,
  );

  return (
    <div>
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th className="w-16">Kz</Th>
            <Th>Bezeichnung</Th>
            <Th numeric className="w-40">
              Bemessungsgrundlage
            </Th>
            <Th numeric className="w-36">
              Steuer
            </Th>
          </Tr>
        </Thead>
        <Tbody>
          {vatReturn.isCorrection && (
            <Tr>
              <Td code>10</Td>
              <Td colSpan={3}>Berichtigte Anmeldung: Kennzahl 10 mit „1“</Td>
            </Tr>
          )}
          {lines.map((line) => (
            <VatRow
              key={line.code}
              line={line}
              expanded={expanded === line.code}
              onToggle={() => setExpanded(expanded === line.code ? null : line.code)}
              numbers={numbers}
              onNavigate={onNavigate}
            />
          ))}
        </Tbody>
      </Table>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <Button variant="quiet" size="sm" onClick={() => setComplete((value) => !value)}>
          {complete ? 'Nur belegte Kennzahlen' : 'Alle Kennzahlen'}
        </Button>
        <Button variant="quiet" size="sm" loading={exporting} onClick={onExport}>
          Als Datei speichern
        </Button>
      </div>
    </div>
  );
};

const VatRow: React.FC<{
  line: VatReturnLine;
  expanded: boolean;
  onToggle: () => void;
  numbers: Map<number, string> | null;
  onNavigate: NavigateFn;
}> = ({ line, expanded, onToggle, numbers, onNavigate }) => {
  const ids = line.entryIds ?? [];
  const payable = line.code === PAYABLE_CODE;
  return (
    <>
      <Tr variant={payable ? 'sum' : 'default'}>
        <Td code>{line.taxCode && line.taxCode !== line.code ? `${line.code} / ${line.taxCode}` : line.code}</Td>
        <Td className="whitespace-normal">
          {ids.length > 0 ? (
            <button
              type="button"
              aria-expanded={expanded}
              onClick={onToggle}
              className="text-left text-ink underline-offset-2 hover:underline"
            >
              {line.label}
            </button>
          ) : (
            line.label
          )}
        </Td>
        <Td numeric className="text-ink-muted">
          {line.hasBase ? formatCents(line.base) : ''}
        </Td>
        <Td numeric>{line.hasTax ? formatCents(line.tax) : ''}</Td>
      </Tr>
      {expanded && ids.length > 0 && (
        <Tr>
          <Td />
          <Td colSpan={3} className="whitespace-normal">
            <span className="text-caption text-ink-subtle mr-2">
              {`${ids.length} ${ids.length === 1 ? 'Buchung' : 'Buchungen'}:`}
            </span>
            {numbers === null ? (
              <span className="text-caption text-ink-subtle">werden gelesen</span>
            ) : (
              ids.map((entryId) => {
                const number = numbers.get(entryId);
                return number ? (
                  <button
                    key={entryId}
                    type="button"
                    className="code-num text-caption text-accent-text mr-3 hover:underline"
                    onClick={() => onNavigate({ area: 'books', tab: 'journal', id: number })}
                  >
                    {number}
                  </button>
                ) : null;
              })
            )}
          </Td>
        </Tr>
      )}
    </>
  );
};

/** Die Zeilen der Zusammenfassenden Meldung: je USt-IdNr. ein Betrag. */
const ZmSheet: React.FC<{ zmReturn: ZMReturn }> = ({ zmReturn }) => (
  <div>
    {(zmReturn.findings ?? []).map((finding) => (
      <Notice key={finding} className="mb-3" text={finding} />
    ))}
    <Table density="kompakt">
      <Thead>
        <Tr>
          <Th className="w-20">Land</Th>
          <Th className="w-48">USt-IdNr.</Th>
          <Th>Kunde</Th>
          <Th numeric className="w-40">
            Sonstige Leistungen
          </Th>
        </Tr>
      </Thead>
      <Tbody>
        {zmReturn.lines.map((line) => (
          <Tr key={line.id || `${line.vatId}-${line.amount}`}>
            <Td code>{line.countryCode}</Td>
            <Td code>{line.vatId}</Td>
            <Td>{line.contactName ?? '—'}</Td>
            <Td numeric>{formatCents(line.amount)}</Td>
          </Tr>
        ))}
        <Tr variant="sum">
          <Td>Summe</Td>
          <Td />
          <Td />
          <Td numeric>{formatCents(zmReturn.totalServices)}</Td>
        </Tr>
      </Tbody>
    </Table>
  </div>
);

