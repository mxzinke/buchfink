import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Api } from '../../services/api';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatDate, parseCents } from '../../utils/formatters';
import { useAnchorScroll, useRouteObject, type NavigateFn } from '../../navigation';
import type {
  Appropriation,
  AppropriationPreview,
  CarryForwardPreview,
  ClosingState,
  ClosingSteps,
  FiscalYearStatus,
  Obligation,
} from '../../types';
import {
  Button,
  ConfirmDialog,
  DetailPanel,
  Dialog,
  Disclosure,
  Field,
  Help,
  Input,
  Notice,
  PageHeader,
  Progress,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  cn,
  toast,
  type Status,
} from '../../components/ui';
import {
  CommitDialog,
  ObligationDecisionDialog,
  ObligationState,
  isManual,
  message,
  obligationKindOf,
  todayISO,
  type CommitTarget,
} from './shared';
import { STEP_VIEWS, StepDetails, StepState, isBookingStep, isStepKey } from './yearSteps';

/**
 * Der Jahresabschluss als Schrittliste: Abschlussbuchungen, Aufstellen,
 * Festschreiben, Feststellen, Ergebnisverwendung, Offenlegen und der
 * Saldenvortrag. Die Abschlussbuchungen bucht Claude; jeder andere Schritt ist
 * eine Handlung der Geschäftsführung oder der Gesellschafter und geschieht nur
 * hier (§ 42a GmbHG, § 146 Abs. 4 AO).
 */

type RowKey = 'bookings' | 'prepared' | 'commit' | 'adopted' | 'appropriation' | 'disclosed' | 'carry';

/** Welche Pflicht des Kalenders zu welchem Schritt gehört. */
const ROW_OF_KIND: Record<string, RowKey> = {
  abschlussbuchungen: 'bookings',
  aufstellung: 'prepared',
  feststellung: 'adopted',
  ergebnisverwendung: 'appropriation',
  offenlegung: 'disclosed',
};

/** Welche Zeile ein Ziel meint, das kein Baustein mit Seitenleiste ist. */
function rowOfTarget(id: string | undefined): RowKey | null {
  if (!id) return null;
  if (id === 'adoption') return 'adopted';
  if (id === 'disclosure') return 'disclosed';
  if (id === 'appropriation') return 'appropriation';
  if (id === 'saldenvortrag') return 'carry';
  if (isStepKey(id)) return 'bookings';
  return ROW_OF_KIND[obligationKindOf(id)] ?? null;
}

const STATUS_BADGE: Record<FiscalYearStatus, Status> = {
  open: 'offen',
  prepared: 'aufgestellt',
  adopted: 'festgestellt',
  disclosed: 'offengelegt',
};

type RowState = 'done' | 'open' | 'waiting' | 'outdated';

const ROW_MARK: Record<RowState, { mark: string; label: string }> = {
  done: { mark: 'bg-positive', label: 'Erledigt' },
  open: { mark: 'bg-attention', label: 'Offen' },
  waiting: { mark: 'bg-ink-faint', label: 'Wartet' },
  outdated: { mark: 'bg-negative', label: 'Überholt' },
};

export interface YearTabProps {
  year: number;
  obligations: Obligation[];
  onObligationsChanged: () => Promise<void>;
  onFiscalYearChanged: () => Promise<void>;
  id?: string;
  onNavigate: NavigateFn;
}

export const YearTab: React.FC<YearTabProps> = ({
  year,
  obligations,
  onObligationsChanged,
  onFiscalYearChanged,
  id,
  onNavigate,
}) => {
  const writeLock = useWriteLock();
  const selection = useRouteObject('closings', 'year');
  const [closing, setClosing] = useState<ClosingState | null>(null);
  const [steps, setSteps] = useState<ClosingSteps | null>(null);
  const [appropriation, setAppropriation] = useState<Appropriation | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [statusDialog, setStatusDialog] = useState<FiscalYearStatus | null>(null);
  const [reopenOpen, setReopenOpen] = useState(false);
  const [commitTarget, setCommitTarget] = useState<CommitTarget | null>(null);
  const [appropriationOpen, setAppropriationOpen] = useState(false);
  const [carryOpen, setCarryOpen] = useState(id === 'saldenvortrag');
  const [stepDecision, setStepDecision] = useState<{ mode: 'skip' | 'reopen'; key: string } | null>(null);
  const [deciding, setDeciding] = useState<Obligation | null>(null);
  const targetRow = rowOfTarget(id);
  const [bookingsOpen, setBookingsOpen] = useState(targetRow === 'bookings');

  useEffect(() => {
    if (targetRow === 'bookings') setBookingsOpen(true);
    if (targetRow === 'carry') setCarryOpen(true);
  }, [targetRow, id]);
  useAnchorScroll(targetRow ? `jahr-${targetRow}` : undefined, id);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [state, nextSteps] = await Promise.all([Api.getClosingState(year), Api.getClosingSteps(year)]);
      setClosing(state);
      setSteps(nextSteps);
      setError('');
      setAppropriation(await Api.getAppropriation(year).catch(() => null));
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  const changed = async () => {
    await load();
    await onObligationsChanged();
    await onFiscalYearChanged();
  };

  const dueOf = useMemo(() => {
    const out: Partial<Record<RowKey, Obligation>> = {};
    for (const o of obligations) {
      const row = ROW_OF_KIND[o.kind];
      if (row && o.fiscalYear === year) out[row] = o;
    }
    return out;
  }, [obligations, year]);
  const sideObligations = obligations.filter((o) => o.kind === 'kapitalertragsteuer');

  if (loading && !closing) return <SkeletonRows rows={8} />;
  if (!closing) {
    return <Notice tone="negative" text={error || 'Der Abschlussstand ließ sich nicht lesen.'} />;
  }

  const fy = closing.fiscalYear;
  const today = todayISO();
  const ended = fy.endDate < today;
  const bookingSteps = (steps?.steps ?? []).filter(isBookingStep);
  const openBookings = bookingSteps.filter((step) => step.state === 'open').length;
  const panelStep =
    selection.id && isStepKey(selection.id) && STEP_VIEWS[selection.id].kind !== 'human'
      ? (steps?.steps ?? []).find((step) => step.key === selection.id) ?? null
      : null;

  const rowState: Record<RowKey, RowState> = {
    bookings: bookingSteps.length > 0 && openBookings === 0 ? 'done' : ended ? 'open' : 'waiting',
    prepared: fy.preparedOn ? 'done' : ended ? 'open' : 'waiting',
    commit: closing.hasYearCommitment ? 'done' : ended ? 'open' : 'waiting',
    adopted: fy.adoptedOn ? 'done' : fy.status === 'prepared' ? 'open' : 'waiting',
    appropriation: appropriation ? 'done' : fy.adoptedOn ? 'open' : 'waiting',
    disclosed: fy.disclosedOn ? 'done' : fy.status === 'adopted' ? 'open' : 'waiting',
    carry: closing.carryForwardCurrent
      ? 'done'
      : closing.carriedForward
        ? 'outdated'
        : ended
          ? 'open'
          : 'waiting',
  };
  const order: RowKey[] = ['prepared', 'commit', 'adopted', 'appropriation', 'disclosed'];
  const next = order.find((row) => rowState[row] === 'open') ?? null;
  const done = (Object.keys(rowState) as RowKey[]).filter((row) => rowState[row] === 'done').length;

  const notEnded = `Das Geschäftsjahr endet am ${formatDate(fy.endDate)}.`;

  return (
    <div>
      <PageHeader
        title={`Jahresabschluss ${year}`}
        context={`${formatDate(fy.startDate)} – ${formatDate(fy.endDate)} · Ergebnis ${formatCents(closing.netIncome)}`}
        helpSummary="Vom Jahresende bis zur Offenlegung: was Claude bucht und was Sie freigeben."
        explain={
          <>
            Die Abschlussbuchungen bucht Claude mit Rückfrage. Aufstellen, Festschreiben,
            Feststellen, Ergebnisverwendung und Offenlegung sind Handlungen der Geschäftsführung und
            der Gesellschafter (§ 242 HGB, § 42a GmbHG, § 325 HGB). Buchfink hält sie mit Datum und
            Nachweis fest. Ab der Feststellung nimmt das Geschäftsjahr keine Buchung mehr an.
          </>
        }
        action={<StatusBadge status={STATUS_BADGE[fy.status]} />}
      />

      {error && <Notice tone="negative" className="mt-6" text={error} />}

      <Progress
        className="mt-6"
        label="Weg zum Abschluss"
        value={Math.round((done / 7) * 100)}
        detail={`${done} von 7 Schritten`}
      />

      {closing.legalReserve?.applies && closing.legalReserve.difference !== 0 && (
        <Notice
          className="mt-6"
          text={`Die gesetzliche Rücklage der UG weicht um ${formatCents(closing.legalReserve.difference)} vom Pflichtbetrag ab.`}
          action={
            <Button
              variant="secondary"
              size="sm"
              disabled={writeLock.locked || closing.hasYearCommitment}
              title={writeLock.hint ?? (closing.hasYearCommitment ? 'Das Jahr ist festgeschrieben.' : undefined)}
              onClick={() =>
                void Api.bookLegalReserve(year)
                  .then(() => toast.success('Gesetzliche Rücklage gebucht.'))
                  .then(load)
                  .catch((e) => toast.error(message(e)))
              }
            >
              Rücklage buchen
            </Button>
          }
        />
      )}

      <ol className="mt-8 flex flex-col">
        <Row
          id="jahr-bookings"
          number={1}
          title="Abschlussbuchungen"
          state={rowState.bookings}
          highlighted={targetRow === 'bookings'}
          caption={
            steps
              ? `${bookingSteps.length - openBookings} von ${bookingSteps.length} Bausteinen erledigt`
              : 'Abschreibung, Abgrenzung, Rückstellungen, Steuern'
          }
          due={dueOf.bookings}
        >
          <Disclosure
            variant="row"
            title="Bausteine"
            open={bookingsOpen}
            onOpenChange={setBookingsOpen}
            className="border-b-0"
          >
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-10" numeric>
                    Nr.
                  </Th>
                  <Th>Baustein</Th>
                  <Th className="w-36">Stand</Th>
                  <Th>Woran</Th>
                </Tr>
              </Thead>
              <Tbody>
                {bookingSteps.map((step) => (
                  <Tr key={step.key} variant={selection.id === step.key ? 'selected' : 'default'}>
                    <Td numeric className="text-ink-subtle">
                      {step.order}
                    </Td>
                    <Td>
                      <button
                        type="button"
                        className="text-left text-ink underline-offset-2 hover:underline"
                        onClick={() => selection.open(step.key)}
                      >
                        {step.label}
                      </button>
                    </Td>
                    <Td>
                      <StepState step={step} />
                    </Td>
                    <Td className="text-ink-muted max-w-[22rem] truncate" title={step.detail || step.reason}>
                      {step.state === 'skipped' ? step.reason || '—' : step.detail || '—'}
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </Disclosure>
        </Row>

        <Row
          id="jahr-prepared"
          number={2}
          title="Abschluss aufstellen"
          state={rowState.prepared}
          highlighted={targetRow === 'prepared'}
          caption={fy.preparedOn ? `Aufgestellt am ${formatDate(fy.preparedOn)}` : 'Die Geschäftsführung unterschreibt den Abschluss.'}
          due={dueOf.prepared}
          action={
            fy.status === 'open' && (
              <Button
                variant={next === 'prepared' ? 'primary' : 'secondary'}
                size="sm"
                disabled={!ended || writeLock.locked}
                title={writeLock.hint ?? (!ended ? notEnded : undefined)}
                onClick={() => setStatusDialog('prepared')}
              >
                Aufstellen
              </Button>
            )
          }
        />

        <Row
          id="jahr-commit"
          number={3}
          title="Jahr festschreiben"
          state={rowState.commit}
          highlighted={targetRow === 'commit'}
          caption={
            closing.hasYearCommitment
              ? `Festgeschrieben bis ${formatDate(closing.committedUntil ?? fy.endDate)}`
              : 'Danach nimmt das Jahr keine Abschlussbuchung mehr an.'
          }
          action={
            !closing.hasYearCommitment && (
              <Button
                variant={next === 'commit' ? 'primary' : 'secondary'}
                size="sm"
                disabled={!ended || writeLock.locked}
                title={writeLock.hint ?? (!ended ? notEnded : undefined)}
                onClick={() =>
                  setCommitTarget({ type: 'year', label: `Geschäftsjahr ${year}`, cutoff: fy.endDate })
                }
              >
                Festschreiben
              </Button>
            )
          }
        />

        <Row
          id="jahr-adopted"
          number={4}
          title="Abschluss feststellen"
          state={rowState.adopted}
          highlighted={targetRow === 'adopted'}
          caption={
            fy.adoptedOn
              ? `Festgestellt am ${formatDate(fy.adoptedOn)}${fy.adoptionNote ? ` · ${fy.adoptionNote}` : ''}`
              : 'Die Gesellschafter beschließen über den Abschluss.'
          }
          due={dueOf.adopted}
          action={
            fy.status === 'prepared' ? (
              <Button
                variant={next === 'adopted' ? 'primary' : 'secondary'}
                size="sm"
                disabled={!closing.canAdopt || writeLock.locked}
                title={writeLock.hint ?? (closing.canAdopt ? undefined : closing.blocker)}
                onClick={() => setStatusDialog('adopted')}
              >
                Feststellen
              </Button>
            ) : fy.adoptedOn ? (
              <Button
                variant="quiet"
                size="sm"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setReopenOpen(true)}
              >
                Zurücksetzen
              </Button>
            ) : undefined
          }
        />

        <Row
          id="jahr-appropriation"
          number={5}
          title="Ergebnisverwendung beschließen"
          state={rowState.appropriation}
          highlighted={targetRow === 'appropriation'}
          caption={
            appropriation
              ? `Beschlossen am ${formatDate(appropriation.decisionDate)} · Ausschüttung ${formatCents(appropriation.distribution)}`
              : 'Gewinnvortrag, Rücklage oder Ausschüttung'
          }
          due={dueOf.appropriation}
          action={
            !appropriation &&
            fy.adoptedOn && (
              <Button
                variant={next === 'appropriation' ? 'primary' : 'secondary'}
                size="sm"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAppropriationOpen(true)}
              >
                Beschluss erfassen
              </Button>
            )
          }
        />

        <Row
          id="jahr-disclosed"
          number={6}
          title="Offenlegen"
          state={rowState.disclosed}
          highlighted={targetRow === 'disclosed'}
          caption={
            fy.disclosedOn
              ? `Offengelegt am ${formatDate(fy.disclosedOn)}${fy.disclosureNote ? ` · ${fy.disclosureNote}` : ''}`
              : 'Einreichung beim Unternehmensregister'
          }
          due={dueOf.disclosed}
          action={
            fy.status === 'adopted' && (
              <Button
                variant={next === 'disclosed' ? 'primary' : 'secondary'}
                size="sm"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setStatusDialog('disclosed')}
              >
                Offenlegung eintragen
              </Button>
            )
          }
        />

        <Row
          id="jahr-carry"
          number={7}
          title={`Saldenvortrag nach ${closing.nextYear}`}
          state={rowState.carry}
          highlighted={targetRow === 'carry'}
          caption={
            closing.carryForwardCurrent
              ? 'Vorgetragen und aktuell'
              : closing.carriedForward
                ? 'Seit dem Vortrag wurde im Jahr noch gebucht.'
                : 'Die Endbestände werden Anfangsbestände des Folgejahres.'
          }
          action={
            <Button variant="quiet" size="sm" onClick={() => setCarryOpen(true)}>
              Vortrag ansehen
            </Button>
          }
        />
      </ol>

      {sideObligations.length > 0 && (
        <section className="mt-8 pt-6 border-t border-line">
          <h3 className="text-label text-ink-muted mb-2">Kapitalertragsteuer auf Ausschüttungen</h3>
          <ul className="flex flex-col">
            {sideObligations.map((o) => (
              <li key={o.key} className="flex items-center gap-4 py-2 border-t border-line first:border-t-0">
                <span className="flex-1 text-body text-ink">{`${o.title} ${o.periodLabel}`}</span>
                <ObligationState obligation={o} />
                {isManual(o) && (
                  <Button
                    variant="quiet"
                    size="sm"
                    disabled={writeLock.locked}
                    title={writeLock.hint}
                    onClick={() => setDeciding(o)}
                  >
                    Erledigt vermerken
                  </Button>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      <DetailPanel
        open={panelStep !== null}
        onOpenChange={(open) => !open && selection.close()}
        title={panelStep?.label ?? ''}
        context={steps ? `Stichtag ${formatDate(steps.cutoff)}` : undefined}
        width="lg"
        footer={
          panelStep && (
            <>
              {(() => {
                const view = STEP_VIEWS[panelStep.key];
                return view.kind !== 'human' && view.route ? (
                  <Button variant="secondary" onClick={() => onNavigate(view.route!)}>
                    {view.routeLabel}
                  </Button>
                ) : null;
              })()}
              {panelStep.state === 'skipped' ? (
                <Button
                  variant="secondary"
                  disabled={writeLock.locked || !steps?.reopenable}
                  title={writeLock.hint ?? steps?.reopenBlocker}
                  onClick={() => setStepDecision({ mode: 'reopen', key: panelStep.key })}
                >
                  Wieder aufnehmen
                </Button>
              ) : panelStep.state === 'open' && !panelStep.automatic ? (
                <Button
                  variant="secondary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setStepDecision({ mode: 'skip', key: panelStep.key })}
                >
                  Überspringen
                </Button>
              ) : null}
            </>
          )
        }
      >
        {panelStep && <StepDetails step={panelStep} year={year} />}
      </DetailPanel>

      <StepDecisionDialog
        decision={stepDecision}
        year={year}
        onClose={() => setStepDecision(null)}
        onDone={(nextSteps) => setSteps(nextSteps)}
      />

      <StatusDialog
        status={statusDialog}
        year={year}
        endDate={fy.endDate}
        onClose={() => setStatusDialog(null)}
        onDone={changed}
      />

      <ReopenDialog open={reopenOpen} year={year} onClose={() => setReopenOpen(false)} onDone={changed} />

      <CommitDialog
        target={commitTarget}
        note="Mit dem Jahr enden auch die Abschlussbuchungen; der Prüfbericht nimmt Abschreibung und Gliederung hinzu."
        onClose={() => setCommitTarget(null)}
        onCommitted={changed}
        onNavigate={onNavigate}
      />

      <AppropriationDialog
        open={appropriationOpen}
        year={year}
        onClose={() => setAppropriationOpen(false)}
        onDone={changed}
      />

      <CarryPanel
        open={carryOpen}
        toYear={closing.nextYear}
        onClose={() => {
          setCarryOpen(false);
          if (id === 'saldenvortrag') selection.close();
        }}
        onDone={changed}
      />

      <ObligationDecisionDialog
        obligation={deciding}
        onClose={() => setDeciding(null)}
        onDone={onObligationsChanged}
      />
    </div>
  );
};

/** Eine Zeile der Schrittliste. */
const Row: React.FC<{
  id: string;
  number: number;
  title: string;
  caption: string;
  state: RowState;
  highlighted?: boolean;
  due?: Obligation;
  action?: React.ReactNode;
  children?: React.ReactNode;
}> = ({ id, number, title, caption, state, highlighted, due, action, children }) => {
  const mark = ROW_MARK[state];
  return (
    <li
      id={id}
      aria-current={highlighted ? 'step' : undefined}
      className={cn(
        'flex items-start gap-4 py-4 border-t border-line first:border-t-0',
        highlighted && 'bg-attention-soft -mx-3 px-3 rounded-control',
      )}
    >
      <span className="w-6 shrink-0 text-caption text-ink-subtle num pt-0.5">{number}</span>
      <div className="flex-1 min-w-0">
        <div className="flex items-start gap-4">
          <div className="flex-1 min-w-0">
            <span className="block text-body font-medium text-ink">{title}</span>
            <span className="block text-caption text-ink-muted mt-0.5">
              {caption}
              {due && state !== 'done' && due.status !== 'overdue' && due.dueDate
                ? ` · fällig am ${formatDate(due.dueDate)}`
                : ''}
            </span>
            {due && state !== 'done' && due.status === 'overdue' && (
              <span className="block mt-1">
                <ObligationState obligation={due} />
              </span>
            )}
          </div>
          <span className="shrink-0 flex items-center gap-2 text-caption text-ink-muted pt-0.5">
            <span className={cn('mark-diamond', mark.mark)} aria-hidden="true" />
            {mark.label}
          </span>
          <span className="shrink-0 w-44 flex justify-end">{action}</span>
        </div>
        {children && <div className="mt-2 -ml-2">{children}</div>}
      </div>
    </li>
  );
};

const STATUS_DIALOG: Record<
  'prepared' | 'adopted' | 'disclosed',
  { title: string; label: string; consequence: string; note?: string; noteHint?: string }
> = {
  prepared: {
    title: 'Abschluss aufstellen',
    label: 'Aufstellen',
    consequence:
      'Die Geschäftsführung bestätigt mit Datum, dass der Abschluss vollständig ist. Gebucht werden kann bis zur Festschreibung weiter.',
  },
  adopted: {
    title: 'Abschluss feststellen',
    label: 'Feststellen',
    consequence:
      'Ab der Feststellung nimmt das Geschäftsjahr keine Buchung mehr an. Zurück geht es nur über die Rücksetzung mit Grund.',
    note: 'Beschluss der Gesellschafter',
    noteHint: 'etwa „Gesellschafterbeschluss vom …"',
  },
  disclosed: {
    title: 'Offenlegung eintragen',
    label: 'Offenlegung eintragen',
    consequence: 'Buchfink reicht nichts ein. Eingetragen wird die Einreichung, die Sie schon erledigt haben.',
    note: 'Auftragsnummer oder Nachweis',
    noteHint: 'aus dem Unternehmensregister',
  },
};

/** Aufstellen, Feststellen und Offenlegen: Datum und, wo nötig, Nachweis. */
const StatusDialog: React.FC<{
  status: FiscalYearStatus | null;
  year: number;
  endDate: string;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ status, year, endDate, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [note, setNote] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const spec = status && status !== 'open' ? STATUS_DIALOG[status] : null;

  useEffect(() => {
    setDate(todayISO());
    setNote('');
    setFieldError('');
    setError('');
  }, [status]);

  async function submit() {
    if (!status || !spec) return;
    if (date < endDate) {
      setFieldError(`Das Datum liegt vor dem Ende des Geschäftsjahres am ${formatDate(endDate)}.`);
      return;
    }
    if (status === 'disclosed' && !note.trim()) {
      setFieldError('Der Nachweis fehlt. Buchfink kennt die Einreichung nur über ihn.');
      return;
    }
    setBusy(true);
    try {
      await Api.setFiscalYearStatus(year, status, date, note.trim());
      toast.success(`${spec.title.replace(' eintragen', '')}: am ${formatDate(date)} vermerkt.`);
      await onDone();
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={spec !== null}
      onOpenChange={(open) => !open && onClose()}
      title={spec ? `${spec.title} ${year}` : ''}
      width="max-w-lg"
      footer={
        <>
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
            {spec?.label}
          </Button>
        </>
      }
    >
      {spec && (
        <div className="flex flex-col gap-4">
          <p className="text-body text-ink">{spec.consequence}</p>
          <Field label="Datum" error={!spec.note ? fieldError || undefined : undefined}>
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          {spec.note && (
            <Field
              label={spec.note}
              hint={spec.noteHint}
              optional={status === 'adopted'}
              error={fieldError || undefined}
            >
              <Input
                value={note}
                onChange={(e) => {
                  setNote(e.target.value);
                  setFieldError('');
                }}
              />
            </Field>
          )}
          {error && <Notice tone="negative" text={error} />}
        </div>
      )}
    </Dialog>
  );
};

/** Die Rücksetzung der Feststellung, mit Grund für das Protokoll. */
const ReopenDialog: React.FC<{
  open: boolean;
  year: number;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ open, year, onClose, onDone }) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Feststellung ${year} zurücksetzen`}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="danger"
            loading={busy}
            onClick={() => {
              if (!reason.trim()) {
                setError('Der Grund fehlt. Er steht danach im Protokoll.');
                return;
              }
              setBusy(true);
              Api.reopenFiscalYear(year, reason.trim())
                .then(onDone)
                .then(() => {
                  setReason('');
                  onClose();
                })
                .catch((e) => setError(message(e)))
                .finally(() => setBusy(false));
            }}
          >
            Zurücksetzen
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <p className="text-body text-ink">
          Das Jahr nimmt danach wieder Buchungen an. Die Festschreibungen bleiben bestehen.
        </p>
        <Field label="Grund" error={error || undefined}>
          <Textarea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
        </Field>
      </div>
    </Dialog>
  );
};

/** Einen Baustein überspringen oder wieder aufnehmen, jeweils mit Grund. */
const StepDecisionDialog: React.FC<{
  decision: { mode: 'skip' | 'reopen'; key: string } | null;
  year: number;
  onClose: () => void;
  onDone: (steps: ClosingSteps) => void;
}> = ({ decision, year, onClose, onDone }) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    setReason('');
    setError('');
  }, [decision]);
  const skip = decision?.mode === 'skip';
  return (
    <Dialog
      open={decision !== null}
      onOpenChange={(open) => !open && onClose()}
      title={skip ? 'Baustein überspringen' : 'Baustein wieder aufnehmen'}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            onClick={() => {
              if (!decision) return;
              if (!reason.trim()) {
                setError('Der Grund fehlt. Er steht danach am Baustein und im Protokoll.');
                return;
              }
              setBusy(true);
              (skip
                ? Api.skipClosingStep(year, decision.key, reason.trim())
                : Api.reopenClosingStep(year, decision.key, reason.trim())
              )
                .then((steps) => {
                  onDone(steps);
                  onClose();
                })
                .catch((e) => setError(message(e)))
                .finally(() => setBusy(false));
            }}
          >
            {skip ? 'Überspringen' : 'Wieder aufnehmen'}
          </Button>
        </>
      }
    >
      <Field
        label="Grund"
        hint={skip ? 'etwa „keine Fremdwährungsposten"' : undefined}
        error={error || undefined}
      >
        <Textarea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Dialog>
  );
};

/**
 * Der Beschluss über die Ergebnisverwendung. Er gehört zum Jahr, dessen
 * Ergebnis verwendet wird, und wird im Folgejahr an seinem Datum gebucht.
 */
const AppropriationDialog: React.FC<{
  open: boolean;
  year: number;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ open, year, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [text, setText] = useState('');
  const [legalReserve, setLegalReserve] = useState('');
  const [otherReserves, setOtherReserves] = useState('');
  const [distribution, setDistribution] = useState('');
  const [preview, setPreview] = useState<AppropriationPreview | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const request = useMemo(
    () => ({
      decisionDate: date,
      text,
      legalReserve: parseCents(legalReserve) ?? 0,
      otherReserves: parseCents(otherReserves) ?? 0,
      distribution: parseCents(distribution) ?? 0,
    }),
    [date, text, legalReserve, otherReserves, distribution],
  );

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewAppropriation(year, request)
        .then((next) => {
          if (cancelled) return;
          setPreview(next);
          setError('');
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setError(message(e));
        });
    }, 300);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [open, year, request]);

  async function book() {
    setBusy(true);
    try {
      await Api.bookAppropriation(year, request);
      toast.success(`Ergebnisverwendung ${year} gebucht.`);
      await onDone();
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const a = preview?.appropriation;
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Ergebnisverwendung ${year}`}
      width="max-w-2xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={!preview || writeLock.locked}
            title={writeLock.hint}
            onClick={() => void book()}
          >
            Beschluss buchen
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-5">
        <StatRow>
          <Stat label="Verwendbar" value={formatCents(preview?.netIncome ?? 0)} />
          <Stat label="Ausschüttung" value={formatCents(a?.distribution ?? 0)} />
          <Stat
            label={
              <>
                Kapitalertragsteuer
                <Help summary="Bei einer Ausschüttung behält die Gesellschaft Steuern ein und führt sie ab." label="Erklärung zur Kapitalertragsteuer">
                  Einbehalten werden 25 % Kapitalertragsteuer und darauf 5,5 % Solidaritätszuschlag (§ 43a
                  Abs. 1 EStG). Die Anmeldung ist bis zum 10. des Folgemonats fällig.
                </Help>
              </>
            }
            value={formatCents((a?.withholdingTax ?? 0) + (a?.solidarityOnWithholding ?? 0))}
          />
          <Stat label="Vortrag" value={formatCents(a?.carryForward ?? 0)} context="bleibt ohne Buchung" />
        </StatRow>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Datum des Beschlusses">
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label="Beschluss" optional>
            <Input value={text} placeholder="Gesellschafterbeschluss vom …" onChange={(e) => setText(e.target.value)} />
          </Field>
          <Field label="Ausschüttung" hint="brutto, vor Steuern">
            <Input align="right" inputMode="decimal" placeholder="0,00" value={distribution} onChange={(e) => setDistribution(e.target.value)} />
          </Field>
          <Field label="Andere Gewinnrücklagen" optional>
            <Input align="right" inputMode="decimal" placeholder="0,00" value={otherReserves} onChange={(e) => setOtherReserves(e.target.value)} />
          </Field>
          <Field label="Zusätzlich in die gesetzliche Rücklage" optional>
            <Input align="right" inputMode="decimal" placeholder="0,00" value={legalReserve} onChange={(e) => setLegalReserve(e.target.value)} />
          </Field>
        </div>
        {(preview?.warnings ?? []).map((warning) => (
          <Notice key={warning} text={warning} />
        ))}
        {error && <Notice tone="negative" text={error} />}
      </div>
    </Dialog>
  );
};

/**
 * Der Saldenvortrag in der Seitenleiste: Vorschau, Prüfung der
 * Bilanzidentität und die Buchung nach Bestätigung.
 */
const CarryPanel: React.FC<{
  open: boolean;
  toYear: number;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ open, toYear, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [preview, setPreview] = useState<CarryForwardPreview | null>(null);
  const [error, setError] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setPreview(null);
    setError('');
    Api.getCarryForwardPreview(toYear)
      .then(setPreview)
      .catch((e) => setError(message(e)));
  }, [open, toYear]);

  const blocked =
    !preview ||
    !preview.isBalanced ||
    Boolean(preview.irreversible) ||
    (preview.alreadyCarried && !preview.needsCorrection) ||
    (preview.entries === 0 && !preview.alreadyCarried);
  const label = preview?.alreadyCarried ? 'Korrekturvortrag buchen' : 'Vortrag buchen';
  const reason = !preview
    ? undefined
    : !preview.isBalanced
      ? `Die Salden ${preview.fromYear} gehen nicht auf.`
      : preview.irreversible
        ? 'Der bestehende Vortrag lässt sich nicht zurücknehmen.'
        : preview.alreadyCarried && !preview.needsCorrection
          ? 'Der Vortrag ist aktuell.'
          : preview.entries === 0
            ? 'Es gibt keine Salden zum Vortrag.'
            : undefined;

  async function carry() {
    if (!preview) return;
    setBusy(true);
    try {
      const entries = await Api.carryForward(preview.toYear);
      toast.success(`Saldenvortrag ${preview.fromYear} nach ${preview.toYear}: ${entries.length} Buchungen.`);
      await onDone();
      setPreview(await Api.getCarryForwardPreview(toYear));
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <DetailPanel
        open={open}
        onOpenChange={(next) => !next && onClose()}
        title={`Saldenvortrag nach ${toYear}`}
        context={preview ? `Buchungsdatum ${formatDate(preview.bookingDate)}` : undefined}
        width="lg"
        footer={
          <Button
            variant="primary"
            loading={busy}
            disabled={blocked || writeLock.locked}
            title={writeLock.hint ?? reason}
            onClick={() => setConfirming(true)}
          >
            {label}
          </Button>
        }
      >
        {error && <Notice tone="negative" className="mb-4" text={error} />}
        {!preview ? (
          !error && <SkeletonRows rows={5} />
        ) : (
          <div className="flex flex-col gap-5">
            <StatRow>
              <Stat label="Jahresergebnis" value={formatCents(preview.netIncome)} />
              <Stat label="Ergebniskonto" value={preview.resultAccount} context={preview.resultAccountName} />
              <Stat
                label="Bilanzidentität"
                value={preview.isBalanced ? 'Ausgeglichen' : formatCents(preview.balanceDifference)}
                tone={preview.isBalanced ? 'positive' : 'negative'}
              />
            </StatRow>
            {preview.needsCorrection && (
              <Notice text="Seit dem letzten Vortrag wurde im Jahr noch gebucht. Ein Korrekturvortrag gleicht das aus." />
            )}
            {preview.priorYearNotCarried && (
              <Notice text={`Das Jahr ${preview.fromYear} hat selbst noch keinen Saldenvortrag.`} />
            )}
            {preview.accrualReleases.length > 0 && (
              <p className="text-body text-ink-muted">
                {`Dazu kommen ${preview.accrualReleases.length} Auflösungen der Rechnungsabgrenzung im Jahr ${preview.toYear}.`}
              </p>
            )}
            <Disclosure title="Konten" context={`${preview.rows.length} Salden, ${preview.entries} Buchungen`} className="mt-0 pt-0 border-t-0">
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th className="w-20">Konto</Th>
                    <Th>Bezeichnung</Th>
                    <Th numeric className="w-32">
                      Schlusssaldo
                    </Th>
                    <Th numeric className="w-32">
                      Vorgetragen
                    </Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {preview.rows.map((row) => (
                    <Tr key={row.account}>
                      <Td code>{row.account}</Td>
                      <Td className="whitespace-normal">{row.name}</Td>
                      <Td numeric>{formatCents(row.closingBalance)}</Td>
                      <Td numeric className="text-ink-muted">
                        {formatCents(row.carried)}
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Disclosure>
          </div>
        )}
      </DetailPanel>
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={label}
        description={
          preview
            ? `${preview.entries} Buchungen zum ${formatDate(preview.bookingDate)} im Geschäftsjahr ${preview.toYear}.` +
              (preview.alreadyCarried ? ' Der bestehende Vortrag wird zuvor per Generalumkehr zurückgenommen.' : '')
            : ''
        }
        confirmLabel={label}
        onConfirm={() => void carry()}
      />
    </>
  );
};
