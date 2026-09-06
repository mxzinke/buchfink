import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Api } from '../services/api';
import { useWriteLock } from './WriteLock';
import { downloadCSV } from '../utils/download';
import { findingParams, findingTarget } from '../utils/findings';
import { formatCents, formatDate } from '../utils/formatters';
import type { MonthCloseState, MonthCloseStep, VatReturn } from '../types';
import type { NavigateFn } from './Sidebar';
import {
  Button,
  Dialog,
  Field,
  HelpPopover,
  Input,
  Notice,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  Textarea,
  cn,
  toast,
} from './ui';

/**
 * Der Monatsabschluss in drei Schritten (Architektur 6.2).
 *
 * Die Reihenfolge ist der Inhalt dieses Dialogs: erst der Prüfbericht, dann die
 * Festschreibung, dann die Bestätigung der Voranmeldung. Was gemeldet wird, muss
 * vorher unveränderbar sein — sonst weicht die Buchführung später von der
 * Meldung ab, und die Abweichung fällt erst dem Prüfer auf.
 *
 * Der Dialog rechnet nichts nach: Stand, Reihenfolge und Kennziffern kommen aus
 * dem Backend (`GetMonthCloseState`, `GetVatReturn`). Alle drei Schritte werden
 * hier ausgeführt — Prüfbericht lesen, festschreiben, Übermittlung bestätigen.
 * Das Kennziffernblatt steht dabei von Anfang an sichtbar da: übertragen wird es
 * in Mein ELSTER, und wer es erst nach der Festschreibung sähe, hätte
 * festgeschrieben, ohne zu wissen, was er meldet.
 *
 * Die Umsatzsteuerseite bleibt der Ort für alles Weitere — Entwürfe,
 * Berichtigungen, Nachträge, die Zusammenfassende Meldung. Der Weg dorthin steht
 * am Ende des dritten Schrittes.
 */

/** Die Kennung des Begründungsfeldes; der Fokus springt beim Absenden dorthin. */
const REASON_FIELD = 'month-close-override-reason';

/** Kennziffer 83 ist die Zahllast; sie steht im Blatt auch dann, wenn sie null ist. */
const PAYABLE_CODE = '83';

/** Was die Festschreibung bedeutet — die Folge, nicht die Aktion (§8.2). */
const COMMIT_CONSEQUENCE =
  'Festgeschriebene Buchungen lassen sich nur noch per Storno korrigieren.';

export interface MonthCloseDialogProps {
  open: boolean;
  /** Der Monat als „JJJJ-MM". */
  month: string;
  /** Die wählbaren Monate; leer heißt: der übergebene Monat steht fest. */
  months?: { value: string; label: string }[];
  onMonthChange?: (month: string) => void;
  onClose: () => void;
  /** Nach der Festschreibung: die aufrufende Ansicht lädt ihren Stand neu. */
  onChanged?: () => void | Promise<void>;
  onNavigate?: NavigateFn;
  /**
   * Ein bereits geladener Stand desselben Monats.
   *
   * `GetMonthCloseState` enthält einen vollständigen Prüflauf. Wer den Dialog
   * aus einer Ansicht öffnet, die den Stand schon zeigt, würde ihn sonst beim
   * Öffnen ein zweites Mal rechnen lassen. Übernommen wird er nur beim Öffnen
   * und nur, wenn er denselben Monat meint; nach jeder Aktion lädt der Dialog
   * selbst neu.
   */
  initialState?: MonthCloseState | null;
}

/** Die Marke vor einem Schritt: erledigt, blockiert, offen oder entfallen. */
const STEP_MARK: Record<MonthCloseStep['state'], { mark: string; label: string }> = {
  done: { mark: 'bg-positive', label: 'Erledigt' },
  blocked: { mark: 'bg-negative', label: 'Blockiert' },
  open: { mark: 'bg-attention', label: 'Offen' },
  not_applicable: { mark: 'bg-ink-faint', label: 'Entfällt' },
};

export const MonthCloseDialog: React.FC<MonthCloseDialogProps> = ({
  open,
  month,
  months = [],
  onMonthChange,
  onClose,
  onChanged,
  onNavigate,
  initialState,
}) => {
  // Festschreiben ändert die Bücher: im Prüfermodus gesperrt, der Stand bleibt
  // lesbar (§10.4).
  const writeLock = useWriteLock();
  const [state, setState] = useState<MonthCloseState | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState('');
  const [committing, setCommitting] = useState(false);
  // Das Kennziffernblatt des Zeitraums, in dem der Monat endet. Es wird eigens
  // geladen: der Stand des Monats sagt, welcher Zeitraum es ist, die Zahlen
  // stehen im Blatt.
  const [vatReturn, setVatReturn] = useState<VatReturn | null>(null);
  const [vatError, setVatError] = useState('');
  const [busy, setBusy] = useState<'export' | 'confirm' | null>(null);
  const [confirmDate, setConfirmDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [confirmTicket, setConfirmTicket] = useState('');
  const [ticketError, setTicketError] = useState('');

  const load = useCallback(
    async (known?: MonthCloseState) => {
      setLoading(true);
      setError('');
      try {
        const next = known ?? (await Api.getMonthCloseState(month));
        setState(next);
        // Das Blatt gehört zum Zeitraum und nicht zum Monat: wer vierteljährlich
        // meldet, sieht es am Quartalsende. Ohne Zeitraum gibt es nichts zu laden.
        if (next.vatApplies && next.vatPeriodKey) {
          try {
            setVatReturn(await Api.getVatReturn(next.vatPeriodKey));
            setVatError('');
          } catch (e) {
            // Ein Blatt, das sich nicht rechnen lässt, hält den Prüfbericht und
            // die Festschreibung nicht auf: der Fehler steht am dritten Schritt.
            setVatReturn(null);
            setVatError(e instanceof Error ? e.message : String(e));
          }
        } else {
          setVatReturn(null);
          setVatError('');
        }
      } catch (e) {
        setState(null);
        setVatReturn(null);
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    },
    [month],
  );

  // Der mitgegebene Stand wird über eine Referenz gelesen: sonst hinge das
  // Öffnen an seiner Identität, und jedes Neuladen der aufrufenden Ansicht
  // stieße hier einen weiteren Prüflauf an.
  const initialRef = useRef(initialState);
  initialRef.current = initialState;

  useEffect(() => {
    if (!open) return;
    const seed = initialRef.current?.month === month ? initialRef.current : undefined;
    void load(seed ?? undefined);
  }, [open, month, load]);

  async function commit() {
    if (!state) return;
    // Der Knopf bleibt aktiv; fehlt die Begründung, steht der Fehler am Feld und
    // der Fokus springt dorthin (§8.3). Ein gesperrter Knopf verschwiege den Grund.
    if (state.blocking > 0 && reason.trim() === '') {
      setReasonError('Blockierende Befunde lassen sich nur mit einer Begründung übergehen.');
      document.getElementById(REASON_FIELD)?.focus();
      return;
    }
    setReasonError('');
    setCommitting(true);
    try {
      const record = await Api.commitPeriod('month', state.label, state.to, reason.trim());
      if (record) {
        toast.success(
          record.timestampStatus === 'confirmed'
            ? `${state.label} festgeschrieben, beglaubigt durch ${record.tsaName}.`
            : `${state.label} festgeschrieben. Der Zeitstempel wird nachgeholt, sobald wieder Netz da ist.`,
        );
      }
      setReason('');
      await load();
      await onChanged?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setCommitting(false);
    }
  }

  /**
   * Das Kennziffernblatt als Datei.
   *
   * Gespeichert wird zuerst: `SaveVatReturn` schreibt den offenen Entwurf auf
   * den heutigen Stand fort. Ohne diesen Schritt gäbe die Datei einen Entwurf
   * von vorgestern aus, während die Ansicht neu gerechnet ist — der Anwender
   * tippte dann andere Kennziffern in Mein ELSTER, als vor ihm stehen.
   */
  async function exportSheet() {
    if (!state?.vatPeriodKey) return;
    setBusy('export');
    try {
      const key = state.vatPeriodKey;
      const saved =
        state.vatStatus === 'submitted' && state.vatReturnId
          ? { id: state.vatReturnId }
          : await Api.saveVatReturn(key);
      downloadCSV(`ustva-${key}.csv`, await Api.exportVatReturnCSV(saved.id));
      toast.success('Kennziffernblatt gespeichert.');
    } catch (e) {
      setVatError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  /**
   * Der dritte Schritt: die Übermittlung bestätigen.
   *
   * Bestätigt wird das Blatt, das auf dem Bildschirm steht; gespeichert wird
   * deshalb zuerst. Das Transferticket ist der Nachweis, dass die Anmeldung
   * angekommen ist — ohne es gibt es keine Bestätigung.
   */
  async function confirmSubmission() {
    if (!state?.vatPeriodKey) return;
    if (confirmTicket.trim() === '') {
      setTicketError('Ohne Transferticket gibt es keine Bestätigung.');
      return;
    }
    setTicketError('');
    setBusy('confirm');
    try {
      const saved = await Api.saveVatReturn(state.vatPeriodKey);
      await Api.confirmVatReturnSubmitted(saved.id, confirmDate, confirmTicket.trim());
      setConfirmTicket('');
      setVatError('');
      await load();
      await onChanged?.();
    } catch (e) {
      setVatError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  /** Führt zu der Stelle, an der ein Befund zu beheben ist, und schließt den Dialog. */
  function jumpTo(objectType: string | undefined, params: ReturnType<typeof findingParams>) {
    const target = findingTarget(objectType);
    if (!target || !onNavigate) return;
    onClose();
    onNavigate(target, params);
  }

  const steps = state?.steps ?? [];
  const findings = state?.findings ?? [];

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title="Monatsabschluss"
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Schließen
          </Button>
          <Button
            variant="primary"
            loading={committing}
            disabled={!state || state.committed || writeLock.locked}
            title={
              writeLock.hint ??
              (state?.committed ? 'Dieser Monat ist bereits festgeschrieben.' : undefined)
            }
            onClick={() => void commit()}
          >
            Monat festschreiben
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-5">
        <div className="flex items-end justify-between gap-4">
          {months.length > 0 && onMonthChange ? (
            <Field label="Monat" className="w-56">
              <Select items={months} value={month} onValueChange={onMonthChange} />
            </Field>
          ) : (
            <p className="text-body text-ink">{state?.label ?? month}</p>
          )}
          <HelpPopover label="Erklärung zum Monatsabschluss">
            Die Reihenfolge ist keine Gewohnheit: Ein Zeitraum, der gemeldet wird, muss vorher
            unveränderbar sein — sonst weicht die Buchführung später von der Meldung ab (§ 146
            Abs. 4 AO, GoBD Rz. 107). Das Kennziffernblatt der Voranmeldung ist vorher schon
            sichtbar; bestätigt wird sie erst nach der Festschreibung, mit dem Transferticket aus
            ELSTER.
          </HelpPopover>
        </div>

        {error && <Notice tone="negative" text={error} />}

        {loading ? (
          <SkeletonRows rows={4} />
        ) : !state ? (
          <p className="text-body text-ink-muted">
            Für diesen Monat ließ sich kein Stand ermitteln.
          </p>
        ) : (
          <>
            {/* Die drei Schritte als Zeilen mit Haarlinie: eine Tabelle wäre
                hier eine Fläche in einer Fläche (§6.3). */}
            <ul className="flex flex-col">
              {steps.map((step) => (
                <li
                  key={step.key}
                  className="flex items-start gap-3 py-3 border-t border-line first:border-t-0"
                >
                  <span className="w-6 shrink-0 text-caption text-ink-subtle num">
                    {step.number}
                  </span>
                  <span className="flex-1 min-w-0">
                    <span className="block text-body text-ink">{step.title}</span>
                    <span className="block text-caption text-ink-muted mt-0.5">{step.note}</span>
                  </span>
                  <span className="shrink-0 flex items-center gap-2 text-caption text-ink-muted">
                    <span
                      className={cn('mark-diamond', STEP_MARK[step.state].mark)}
                      aria-hidden="true"
                    />
                    {STEP_MARK[step.state].label}
                  </span>
                </li>
              ))}
            </ul>

            {findings.length > 0 && (
              <div>
                <h3 className="text-label text-ink-muted">
                  Befunde bis {formatDate(state.to)}
                </h3>
                <ul className="flex flex-col mt-2">
                  {findings.map((finding) => (
                    <li
                      key={`${finding.rule}-${finding.objectId}-${finding.message}`}
                      className="flex items-start gap-3 py-2 border-t border-line first:border-t-0"
                    >
                      <span
                        className={cn(
                          'w-24 shrink-0 text-caption',
                          finding.severity === 'blocking'
                            ? 'text-negative-text'
                            : 'text-attention-text',
                        )}
                      >
                        {finding.severity === 'blocking' ? 'Blockierend' : 'Hinweis'}
                      </span>
                      <span className="flex-1 text-body text-ink-muted">{finding.message}</span>
                      {onNavigate && findingTarget(finding.objectType) && (
                        <Button
                          variant="quiet"
                          size="sm"
                          className="shrink-0 -my-1"
                          onClick={() => jumpTo(finding.objectType, findingParams(finding))}
                        >
                          Hin dazu
                        </Button>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <p className="text-body text-ink-muted">{COMMIT_CONSEQUENCE}</p>

            {state.blocking > 0 && !state.committed && (
              <Field
                label="Begründung für das Übergehen"
                hint="steht am Prüflauf und im Protokoll"
                error={reasonError || undefined}
              >
                <Textarea
                  id={REASON_FIELD}
                  value={reason}
                  onChange={(e) => {
                    setReason(e.target.value);
                    if (reasonError) setReasonError('');
                  }}
                />
              </Field>
            )}

            {/* Der dritte Schritt steht vollständig hier: Kennziffernblatt,
                Datei und Bestätigung mit Transferticket. Das Blatt ist von
                Anfang an sichtbar; bestätigt wird erst nach der Festschreibung
                (Architektur 6.2). */}
            {state.vatApplies && (
              <div className="flex flex-col gap-3">
                <div className="flex items-baseline justify-between gap-4">
                  <h3 className="text-label text-ink-muted">
                    {`Kennziffernblatt ${state.vatPeriodLabel ?? ''}`}
                  </h3>
                  <span className="text-caption text-ink-subtle">
                    {state.vatStatus === 'submitted'
                      ? `übermittelt am ${formatDate(state.vatSubmittedAt ?? '')}`
                      : state.vatDueDate
                        ? `fällig am ${formatDate(state.vatDueDate)}`
                        : ''}
                  </span>
                </div>

                {vatError && <Notice tone="negative" text={vatError} />}

                {vatReturn === null ? (
                  <p className="text-body text-ink-muted">
                    Für diesen Zeitraum liegt noch kein Kennziffernblatt vor.
                  </p>
                ) : (
                  <Table density="kompakt">
                    <Thead>
                      <Tr>
                        <Th className="w-16">Kz</Th>
                        <Th>Position</Th>
                        <Th numeric className="w-36">
                          Bemessungsgrundlage
                        </Th>
                        <Th numeric className="w-32">
                          Steuer
                        </Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {/* Nur die belegten Kennziffern und die Zahllast: der
                          vollständige Vordruck steht auf der
                          Umsatzsteuerseite. */}
                      {(vatReturn.figures ?? [])
                        .filter(
                          (line) =>
                            line.base !== 0 || line.tax !== 0 || line.code === PAYABLE_CODE,
                        )
                        .map((line) => (
                          <Tr key={line.code}>
                            <Td code>{line.code}</Td>
                            <Td className="whitespace-normal text-ink-muted">{line.label}</Td>
                            <Td numeric className="text-ink-muted">
                              {line.hasBase ? formatCents(line.base) : '—'}
                            </Td>
                            <Td numeric>{line.hasTax ? formatCents(line.tax) : '—'}</Td>
                          </Tr>
                        ))}
                    </Tbody>
                  </Table>
                )}

                <div className="flex flex-wrap items-end gap-4">
                  <Button
                    variant="secondary"
                    size="sm"
                    loading={busy === 'export'}
                    disabled={busy !== null || !state.vatPeriodKey}
                    onClick={() => void exportSheet()}
                  >
                    Kennziffernblatt als Datei
                  </Button>
                  {onNavigate && (
                    <Button
                      variant="quiet"
                      size="sm"
                      onClick={() => {
                        onClose();
                        onNavigate('vat');
                      }}
                    >
                      Zur Voranmeldung
                    </Button>
                  )}
                </div>

                {state.vatStatus === 'submitted' ? (
                  <p className="text-body text-ink-muted">
                    {`Die Übermittlung ist bestätigt${
                      state.vatSubmittedAt ? ` — am ${formatDate(state.vatSubmittedAt ?? '')}` : ''
                    }. Eine Änderung führt über die berichtigte Anmeldung auf der Umsatzsteuerseite.`}
                  </p>
                ) : !state.committed ? (
                  <p className="text-body text-ink-muted">
                    Bestätigt wird die Übermittlung nach der Festschreibung: gemeldet wird ein
                    Stand, der sich nicht mehr ändert.
                  </p>
                ) : (
                  <div className="flex flex-wrap items-end gap-4">
                    <Field label="Datum der Übermittlung" className="w-48">
                      <Input
                        type="date"
                        value={confirmDate}
                        onChange={(e) => setConfirmDate(e.target.value)}
                      />
                    </Field>
                    <Field
                      label="Transferticket"
                      error={ticketError || undefined}
                      hint="aus Mein ELSTER"
                      className="w-64"
                    >
                      <Input
                        className="code-num"
                        value={confirmTicket}
                        onChange={(e) => {
                          setConfirmTicket(e.target.value);
                          if (ticketError) setTicketError('');
                        }}
                      />
                    </Field>
                    <Button
                      variant="secondary"
                      loading={busy === 'confirm'}
                      disabled={busy !== null || writeLock.locked}
                      title={writeLock.hint}
                      onClick={() => void confirmSubmission()}
                    >
                      Übermittlung bestätigen
                    </Button>
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </Dialog>
  );
};
