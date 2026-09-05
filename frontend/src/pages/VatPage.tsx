import React, { useEffect, useMemo, useState } from 'react';
import { ChevronDown, ChevronRight, Download, FileCheck2, Save } from 'lucide-react';
import type { NavigateFn } from '../components/Sidebar';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { downloadCSV } from '../utils/download';
import { formatCents, formatDate } from '../utils/formatters';
import type {
  JournalEntry,
  VatPeriodStatus,
  VatReturn,
  VatReturnLine,
  ZMPeriodStatus,
  ZMReturn,
} from '../types';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  Switch,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Umsatzsteuer: Voranmeldung und Zusammenfassende Meldung.
 *
 * Buchfink übermittelt nicht selbst. Diese Ansicht erzeugt das Kennziffernblatt
 * und die Meldedatei, der Anwender gibt beides in Mein ELSTER bzw. im
 * BZSt-Online-Portal ein und bestätigt die Übermittlung hier mit dem
 * Transferticket. Gerechnet wird nichts davon hier: die Kennziffern, ihre
 * Reihenfolge, die Zahllast und die Nachträge kommen fertig aus dem Backend
 * (`GetVatReturn`), und die Buchungen hinter einer Kennziffer stehen als
 * Drill-down an der Zeile.
 */

/**
 * Kennziffer 83 ist die Zahllast: die Summenzeile des Vordrucks. Sie steht
 * deshalb auch dann im Blatt, wenn sie null ist — eine Voranmeldung ohne
 * Zahllast ist eine Aussage und keine Auslassung.
 */
const PAYABLE_CODE = '83';

/** Wonach die Bestätigung fragt — für beide Meldungen dieselbe Maske. */
interface ConfirmTarget {
  kind: 'vat' | 'zm';
  periodKey: string;
  label: string;
}

export interface VatPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile. Die Zeiträume folgen ihm. */
  year: number;
  /** Weg von der Buchungsnummer im Drill-down ins Journal (UST-01). */
  onNavigate?: NavigateFn;
}

export const VatPage: React.FC<VatPageProps> = ({ year, onNavigate }) => {
  // Entwurf, Bestätigung und Berichtigung schreiben; Rechnen und Ausgeben
  // bleiben im Prüfermodus möglich (§10.4).
  const writeLock = useWriteLock();
  const [periods, setPeriods] = useState<VatPeriodStatus[]>([]);
  const [periodKey, setPeriodKey] = useState<string>('');
  const [vatReturn, setVatReturn] = useState<VatReturn | null>(null);
  const [savedReturns, setSavedReturns] = useState<VatReturn[]>([]);

  const [zmPeriods, setZmPeriods] = useState<ZMPeriodStatus[]>([]);
  const [zmKey, setZmKey] = useState<string>('');
  const [zmReturn, setZmReturn] = useState<ZMReturn | null>(null);
  const [savedZM, setSavedZM] = useState<ZMReturn[]>([]);

  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [completeForm, setCompleteForm] = useState(false);

  const [loading, setLoading] = useState(true);
  const [loadingReturn, setLoadingReturn] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const [confirming, setConfirming] = useState<ConfirmTarget | null>(null);
  const [confirmDate, setConfirmDate] = useState(today());
  const [confirmTicket, setConfirmTicket] = useState('');
  const [confirmNote, setConfirmNote] = useState('');
  // Was das Backend an der Bestätigung ablehnt, bleibt im Dialog stehen: ein
  // Toast wäre weg, bevor der Anwender das Feld gefunden hat (§10.4).
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [ticketError, setTicketError] = useState<string | null>(null);

  useEffect(() => {
    void loadYear();
  }, [year]);

  useEffect(() => {
    if (periodKey) void loadReturn(periodKey);
  }, [periodKey]);

  useEffect(() => {
    if (zmKey) void loadZM(zmKey);
  }, [zmKey]);

  async function loadYear() {
    setLoading(true);
    try {
      const [vatPeriods, returns, zmList, zmReturns, journal] = await Promise.all([
        Api.getVatPeriods(year),
        Api.getVatReturns(year),
        Api.getZMPeriods(year),
        Api.getZMReturns(year),
        Api.getAllJournalEntries(),
      ]);
      setPeriods(vatPeriods);
      setSavedReturns(returns);
      setZmPeriods(zmList);
      setSavedZM(zmReturns);
      setEntries(journal);
      setError(null);
      setPeriodKey((current) => pickPeriod(vatPeriods, current));
      setZmKey((current) => pickPeriod(zmList, current));
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }

  /** Der Entwurf wird jedes Mal neu gerechnet — er ist der Stand von heute. */
  async function loadReturn(key: string) {
    setLoadingReturn(true);
    setExpanded(null);
    try {
      setVatReturn(await Api.getVatReturn(key));
      setError(null);
    } catch (e) {
      setVatReturn(null);
      setError(message(e));
    } finally {
      setLoadingReturn(false);
    }
  }

  async function loadZM(key: string) {
    try {
      setZmReturn(await Api.getZMReturn(key));
    } catch (e) {
      setZmReturn(null);
      toast.error(message(e));
    }
  }

  /** Nach jeder Änderung: Stand der Zeiträume und Listen neu lesen. */
  async function refreshLists() {
    const [vatPeriods, returns, zmList, zmReturns] = await Promise.all([
      Api.getVatPeriods(year),
      Api.getVatReturns(year),
      Api.getZMPeriods(year),
      Api.getZMReturns(year),
    ]);
    setPeriods(vatPeriods);
    setSavedReturns(returns);
    setZmPeriods(zmList);
    setSavedZM(zmReturns);
  }

  const selectedPeriod = periods.find((p) => p.key === periodKey);
  const selectedZmPeriod = zmPeriods.find((p) => p.key === zmKey);

  const entryById = useMemo(() => {
    const map = new Map<number, JournalEntry>();
    for (const entry of entries) map.set(entry.id, entry);
    return map;
  }, [entries]);

  // Die Meldung gibt es nur, wo es innergemeinschaftliche Umsätze gibt
  // (§ 18a Abs. 1 Satz 1 UStG: „soweit … ausgeführt").
  const hasZM = zmPeriods.some((p) => p.total !== 0 || p.returnId) || savedZM.length > 0;

  async function handleSaveDraft() {
    if (!periodKey) return;
    setBusy('save');
    try {
      setVatReturn(await Api.saveVatReturn(periodKey));
      await refreshLists();
      toast.success('Voranmeldung als Entwurf gespeichert.');
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  /**
   * Die Datei setzt eine gespeicherte Anmeldung voraus: ausgegeben wird, was
   * abgelegt ist, nicht eine Rechnung, die niemand wiederfindet.
   *
   * Mit `id` ist genau diese Anmeldung gemeint — so steht es in der Liste der
   * gespeicherten Voranmeldungen. Ohne `id` ist die laufende gemeint, und dann
   * wird zuerst gespeichert: `SaveVatReturn` schreibt den offenen Entwurf auf
   * den heutigen Stand fort. Ohne diesen Schritt gäbe die Datei einen Entwurf
   * von vorgestern aus, während die Ansicht neu gerechnet ist — der Anwender
   * tippte dann andere Kennziffern in Mein ELSTER, als vor ihm auf dem
   * Bildschirm stehen. Anders beim übermittelten Zeitraum: dort ist die
   * bestätigte Anmeldung das Protokoll, und einen zweiten Entwurf legt das
   * Backend ohnehin nicht an.
   */
  async function handleExportVat(key: string, id?: number) {
    setBusy(`export-${id ?? key}`);
    try {
      const period = periods.find((p) => p.key === key);
      const target =
        id ??
        (period?.status === 'submitted' && period.returnId
          ? period.returnId
          : (await Api.saveVatReturn(key)).id);
      const csv = await Api.exportVatReturnCSV(target);
      downloadCSV(`ustva-${key}.csv`, csv);
      await refreshLists();
      toast.success('Kennziffernblatt gespeichert.');
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  /** Die Meldedatei folgt derselben Regel wie das Kennziffernblatt. */
  async function handleExportZM(key: string, id?: number) {
    setBusy(`zm-export-${id ?? key}`);
    try {
      const period = zmPeriods.find((p) => p.key === key);
      const target =
        id ??
        (period?.status === 'submitted' && period.returnId
          ? period.returnId
          : (await Api.saveZMReturn(key)).id);
      const csv = await Api.exportZMCSV(target);
      downloadCSV(`zm-${key}.csv`, csv);
      await refreshLists();
      toast.success('Meldedatei gespeichert.');
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  /** Der Entwurf der Zusammenfassenden Meldung — wie bei der Voranmeldung. */
  async function handleSaveZMDraft() {
    if (!zmKey) return;
    setBusy('zm-save');
    try {
      await Api.saveZMReturn(zmKey);
      await refreshLists();
      await loadZM(zmKey);
      toast.success('Zusammenfassende Meldung als Entwurf gespeichert.');
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  async function handleCorrection(key: string) {
    setBusy(`correction-${key}`);
    try {
      const created = await Api.createVatCorrection(key);
      await refreshLists();
      // Der Effekt auf periodKey feuert nicht, wenn der berichtigte Zeitraum
      // schon gewählt ist. Das Blatt wird deshalb hier geladen — sonst bliebe
      // die Ansicht auf dem Stand vor der Berichtigung stehen.
      setPeriodKey(created.periodKey);
      await loadReturn(created.periodKey);
      toast.success(`Berichtigte Voranmeldung für ${key} angelegt.`);
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  /**
   * Die Berichtigung der Zusammenfassenden Meldung (§ 18a Abs. 10 UStG): der
   * Weg für einen Umsatz, dessen Meldezeitraum schon übermittelt ist. Auch hier
   * meldet die Berichtigung den Zeitraum vollständig neu, sie ist keine
   * Differenz.
   */
  async function handleZMCorrection(key: string) {
    setBusy(`zm-correction-${key}`);
    try {
      const created = await Api.createZMCorrection(key);
      await refreshLists();
      // Wie bei der Voranmeldung: steht der berichtigte Zeitraum schon in der
      // Auswahl, feuert der Effekt nicht, und die Ansicht bliebe auf dem Stand
      // vor der Berichtigung stehen.
      setZmKey(created.periodKey);
      await loadZM(created.periodKey);
      toast.success(`Berichtigte Zusammenfassende Meldung für ${key} angelegt.`);
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy(null);
    }
  }

  function openConfirm(target: ConfirmTarget) {
    setConfirmDate(today());
    setConfirmTicket('');
    setConfirmNote('');
    setConfirmError(null);
    setTicketError(null);
    setConfirming(target);
  }

  async function handleConfirm() {
    if (!confirming) return;
    const target = confirming;
    // Das Transferticket ist der Nachweis, dass die Anmeldung angekommen ist.
    // Sein Fehlen wird am Feld gemeldet, wo es zu beheben ist.
    if (confirmTicket.trim() === '') {
      setTicketError('Ohne Transferticket gibt es keine Bestätigung.');
      return;
    }
    setTicketError(null);
    setConfirmError(null);
    setBusy('confirm');
    try {
      // Bestätigt wird das Blatt, das auf dem Bildschirm steht. Gespeichert wird
      // deshalb zuerst: Save schreibt den offenen Entwurf auf den heutigen Stand
      // fort. Ein älterer Entwurf würde vom Backend ohnehin abgelehnt
      // (ensureCurrent) — nur müsste der Anwender dann selbst darauf kommen,
      // erst „Entwurf speichern" zu drücken.
      if (target.kind === 'vat') {
        const saved = await Api.saveVatReturn(target.periodKey);
        await Api.confirmVatReturnSubmitted(saved.id, confirmDate, confirmTicket, confirmNote);
      } else {
        const saved = await Api.saveZMReturn(target.periodKey);
        await Api.confirmZMSubmitted(saved.id, confirmDate, confirmTicket, confirmNote);
      }
      setConfirming(null);
      await refreshLists();
      if (target.kind === 'vat') await loadReturn(target.periodKey);
      else await loadZM(target.periodKey);
      // Kein Toast: der sich schließende Dialog ist die Rückmeldung, und der
      // neue Stand steht in der Liste (§8.5).
    } catch (e) {
      setConfirmError(message(e));
    } finally {
      setBusy(null);
    }
  }

  // Listen aus dem Backend werden hier auf einen Wert gebracht, bevor eine
  // Länge davon abhängt: Go serialisiert eine leere Liste als `null`, und ein
  // `null.length` im Rendern nimmt die ganze Ansicht mit — im Regelfall, denn
  // der Regelfall ist der Zeitraum ohne Nachtrag.
  const figures = vatReturn?.figures ?? [];
  const lateEntries = vatReturn?.lateEntries ?? [];
  const zmLines = zmReturn?.lines ?? [];
  const zmLateEntries = zmReturn?.lateEntries ?? [];
  const visibleFigures = completeForm
    ? figures
    : figures.filter((line) => line.base !== 0 || line.tax !== 0 || line.code === PAYABLE_CODE);

  // Die Anmeldung, die diese Berichtigung ersetzt. Ohne sie bliebe „berichtigt"
  // eine Behauptung ohne Bezug.
  const corrected = vatReturn?.correctsId
    ? savedReturns.find((r) => r.id === vatReturn.correctsId)
    : undefined;

  const payable = vatReturn?.payable ?? 0;
  const refund = payable < 0;
  // Ohne gerechnetes Blatt gibt es nichts zu speichern und nichts auszugeben.
  const sheetHint = 'Für diesen Zeitraum liegt noch kein Kennziffernblatt vor.';
  // Ohne Festschreibung ist die Bestätigung gesperrt: eine übermittelte
  // Anmeldung muss sich auf einen unveränderlichen Stand stützen.
  const canConfirm = Boolean(selectedPeriod?.committed) && selectedPeriod?.status !== 'submitted';
  // Die Meldung wird an denselben Bedingungen gesperrt wie die Voranmeldung:
  // ohne Festschreibung weist das Backend sie ab (ZMService.ensureCommitted),
  // und ohne USt-IdNr. am Abnehmer nimmt das Portal sie nicht an.
  const zmSheetHint = 'Für diesen Meldezeitraum liegt noch keine Meldung vor.';
  const zmConfirmHint =
    (zmReturn?.findings?.length ?? 0) > 0
      ? 'Die Meldung ist unvollständig; das Portal nimmt sie so nicht an.'
      : !selectedZmPeriod?.committed
        ? 'Der Meldezeitraum ist noch nicht festgeschrieben. Das geschieht unter Steuerfristen.'
        : selectedZmPeriod?.status === 'submitted'
          ? 'Für diesen Meldezeitraum ist die Übermittlung bereits bestätigt.'
          : undefined;

  const confirmHint = !selectedPeriod?.committed
    ? 'Der Zeitraum ist noch nicht festgeschrieben. Das geschieht unter Steuerfristen.'
    : selectedPeriod?.status === 'submitted'
      ? 'Für diesen Zeitraum ist die Übermittlung bereits bestätigt.'
      : undefined;

  if (loading) {
    return (
      <div className="max-w-[1200px] mx-auto px-8 py-8">
        <SkeletonRows rows={10} />
      </div>
    );
  }

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Umsatzsteuer"
        context="Voranmeldung und Zusammenfassende Meldung"
        action={
          <div className="flex gap-2">
            <Button
              variant="secondary"
              icon={<Save className="w-4 h-4" strokeWidth={1.5} />}
              loading={busy === 'save'}
              disabled={!vatReturn || busy !== null || writeLock.locked}
              title={writeLock.hint ?? (!vatReturn ? sheetHint : undefined)}
              onClick={() => void handleSaveDraft()}
            >
              Entwurf speichern
            </Button>
            <Button
              variant="secondary"
              icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
              loading={busy === `export-${periodKey}`}
              disabled={!vatReturn || busy !== null}
              title={!vatReturn ? sheetHint : undefined}
              onClick={() => void handleExportVat(periodKey)}
            >
              Als Datei
            </Button>
            <Button
              variant="primary"
              icon={<FileCheck2 className="w-4 h-4" strokeWidth={1.5} />}
              disabled={!canConfirm || busy !== null || writeLock.locked}
              title={writeLock.hint ?? confirmHint}
              onClick={() =>
                openConfirm({
                  kind: 'vat',
                  periodKey,
                  label: `Voranmeldung ${selectedPeriod?.label ?? periodKey}`,
                })
              }
            >
              Als übermittelt bestätigen
            </Button>
          </div>
        }
      />

      {error && (
        <Notice
          className="mt-6"
          tone="negative"
          text={error}
          action={
            <Button variant="secondary" size="sm" onClick={() => void loadYear()}>
              Erneut rechnen
            </Button>
          }
        />
      )}

      <Section
        title="Voranmeldung"
        context={selectedPeriod ? `${selectedPeriod.label} · fällig am ${formatDate(selectedPeriod.dueDate)}` : undefined}
        divider={false}
        className="mt-6"
        action={
          <Select
            items={periods.map((p) => ({ value: p.key, label: p.label }))}
            value={periodKey}
            onValueChange={(next) => setPeriodKey(String(next))}
            className="w-56"
          />
        }
      >
        {selectedPeriod && (
          <StatRow>
            <Stat
              label={refund ? 'Erstattungsanspruch' : 'Zahllast'}
              value={formatCents(Math.abs(payable))}
              context={refund ? 'Guthaben beim Finanzamt' : 'an das Finanzamt zu zahlen'}
              tone={refund ? 'positive' : 'neutral'}
            />
            <Stat
              label="Fällig am"
              value={formatDate(selectedPeriod.dueDate)}
              context={selectedPeriod.isOverdue ? 'überfällig' : 'nach § 18 Abs. 1 UStG'}
              tone={selectedPeriod.isOverdue ? 'negative' : 'neutral'}
            />
            <Stat
              label="Festschreibung"
              value={selectedPeriod.committed ? 'Erfolgt' : 'Offen'}
              context={selectedPeriod.committed ? 'Stand unveränderlich' : 'unter Steuerfristen'}
            />
            <Stat
              label="Stand"
              value={selectedPeriod.status === 'submitted' ? 'Übermittelt' : 'Entwurf'}
              context={
                selectedPeriod.submittedAt
                  ? `bestätigt am ${formatDate(selectedPeriod.submittedAt)}`
                  : 'noch nicht bestätigt'
              }
            />
          </StatRow>
        )}
      </Section>

      <Section
        title="Kennziffern des Vordrucks USt 1 A"
        context={vatReturn?.isCorrection ? 'Berichtigte Anmeldung · Kennziffer 10 gesetzt' : undefined}
        action={
          <div className="flex items-center gap-4">
            <Switch
              checked={completeForm}
              onCheckedChange={(next) => setCompleteForm(next)}
              label="Vollständiger Vordruck"
            />
            <HelpPopover label="Erklärung zu den Kennziffern">
              Der Vordruck trägt die gebuchte Steuer, nicht die aus der Bemessungsgrundlage
              nachgerechnete: die Rundung je Rechnung ist die richtige. Weicht beides voneinander
              ab, steht die Differenz in der Spalte Abweichung. Bemessungsgrundlagen führt der
              Vordruck in vollen Euro.
            </HelpPopover>
          </div>
        }
      >
        {loadingReturn ? (
          <SkeletonRows rows={8} />
        ) : visibleFigures.length === 0 ? (
          <EmptyState
            title="Keine Umsätze in diesem Zeitraum"
            description="Sobald Buchungen mit Steuerschlüssel in den Zeitraum fallen, füllt sich das Blatt."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-24">Kz</Th>
                <Th>Position</Th>
                <Th numeric className="w-40">
                  Bemessungsgrundlage
                </Th>
                <Th numeric className="w-36">
                  Steuer
                </Th>
                <Th numeric className="w-32">
                  Abweichung
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {/* Kennziffer 10 ist das Merkmal, an dem das Finanzamt die
                  ersetzende Anmeldung erkennt (§ 153 AO). Sie steht deshalb im
                  Blatt und nicht nur in der Ausgabedatei. */}
              {vatReturn?.isCorrection && (
                <Tr>
                  <Td code>10</Td>
                  <Td colSpan={4} className="whitespace-normal">
                    Berichtigte Anmeldung — Kennziffer 10 mit „1“ belegt
                    {corrected?.submittedAt
                      ? ` · ersetzt die Anmeldung vom ${formatDate(corrected.submittedAt)}`
                      : ''}
                  </Td>
                </Tr>
              )}
              {visibleFigures.map((line) => (
                <FigureRow
                  key={line.code}
                  line={line}
                  expanded={expanded === line.code}
                  onToggle={() => setExpanded(expanded === line.code ? null : line.code)}
                  entryById={entryById}
                  onNavigate={onNavigate}
                />
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {lateEntries.length > 0 && (
        <Section
          title="Nachträge zu übermittelten Zeiträumen"
          context={`${lateEntries.length} Buchungen in bereits übermittelten Zeiträumen`}
          action={
            <HelpPopover label="Erklärung zu den Nachträgen">
              Diese Buchungen gehören in einen Zeitraum, dessen Voranmeldung bereits übermittelt
              ist. Buchfink schiebt sie nicht stillschweigend in den laufenden Zeitraum, denn
              § 18 Abs. 1 UStG ordnet den Umsatz dem Zeitraum zu, in dem er entstanden ist. Der
              Weg ist die berichtigte Anmeldung des ursprünglichen Zeitraums.
            </HelpPopover>
          }
        >
          <Table>
            <Thead>
              <Tr>
                <Th className="w-32">Buchung</Th>
                <Th className="w-28">Datum</Th>
                <Th>Text</Th>
                <Th className="w-28">Zeitraum</Th>
                <Th className="w-16">Kz</Th>
                <Th numeric className="w-32">
                  Bemessung
                </Th>
                <Th numeric className="w-28">
                  Steuer
                </Th>
                <Th className="w-48" aria-label="Aktion" />
              </Tr>
            </Thead>
            <Tbody>
              {lateEntries.map((late) => (
                <Tr key={`${late.entryId}-${late.code}`}>
                  <Td code>
                    <EntryLink
                      entryNumber={late.entryNumber}
                      onNavigate={onNavigate}
                    />
                  </Td>
                  <Td className="num text-ink-subtle">{formatDate(late.bookingDate)}</Td>
                  <Td className="max-w-[22rem] truncate" title={late.description}>
                    {late.description}
                  </Td>
                  <Td className="code-num text-caption text-ink-muted">{late.periodKey}</Td>
                  <Td code>{late.code}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(late.base)}
                  </Td>
                  <Td numeric>{formatCents(late.tax)}</Td>
                  <Td className="text-right">
                    <Button
                      variant="secondary"
                      size="sm"
                      loading={busy === `correction-${late.periodKey}`}
                      disabled={busy !== null || writeLock.locked}
                      title={writeLock.hint}
                      onClick={() => void handleCorrection(late.periodKey)}
                    >
                      Berichtigung erzeugen
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Section>
      )}

      <Section
        title="Gespeicherte Voranmeldungen"
        context={`Geschäftsjahr ${year} · Entwürfe und Übermittlungsprotokoll`}
      >
        {savedReturns.length === 0 ? (
          <EmptyState
            title="Noch keine Voranmeldung gespeichert"
            description="Ein Entwurf entsteht über den Knopf oben; bestätigt wird er nach der Eingabe in Mein ELSTER."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-32">Zeitraum</Th>
                <Th className="w-28">Fällig</Th>
                <Th numeric className="w-36">
                  Zahllast
                </Th>
                <Th className="w-40">Stand</Th>
                <Th>Transferticket</Th>
                <Th className="w-28" aria-label="Aktion" />
              </Tr>
            </Thead>
            <Tbody>
              {[...savedReturns]
                .sort((a, b) => b.periodKey.localeCompare(a.periodKey) || b.id - a.id)
                .map((r) => (
                  <Tr key={r.id}>
                    <Td>
                      <span className="flex items-center gap-2">
                        <span className="code-num text-caption text-ink-muted">{r.periodKey}</span>
                        {r.isCorrection && (
                          <span className="text-caption text-attention-text">Berichtigung</span>
                        )}
                      </span>
                    </Td>
                    <Td className="num text-ink-subtle">{formatDate(r.dueDate ?? '')}</Td>
                    <Td numeric className={r.payable < 0 ? 'text-positive-text' : undefined}>
                      {formatCents(r.payable)}
                    </Td>
                    {/* „Übermittelt" ist kein Zustand aus dem Vokabular des
                        Design-Konzepts (§11.3) und bekommt deshalb kein
                        Abzeichen: es steht als Wort. */}
                    <Td className={r.status === 'submitted' ? 'text-ink' : 'text-ink-subtle'}>
                      {r.status === 'submitted' ? 'Übermittelt' : 'Entwurf'}
                    </Td>
                    <Td className="code-num text-caption text-ink-muted">
                      {r.transferTicket
                        ? `${r.transferTicket} · ${formatDate(r.submittedAt ?? '')}`
                        : '—'}
                    </Td>
                    <Td className="text-right">
                      <Button
                        variant="quiet"
                        size="sm"
                        loading={busy === `export-${r.id}`}
                        disabled={busy !== null}
                        onClick={() => void handleExportVat(r.periodKey, r.id)}
                      >
                        Als Datei
                      </Button>
                    </Td>
                  </Tr>
                ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {hasZM && (
        <>
          <Section
            title="Zusammenfassende Meldung"
            context={
              selectedZmPeriod
                ? `${selectedZmPeriod.label} · fällig am ${formatDate(selectedZmPeriod.dueDate)}`
                : undefined
            }
            action={
              <div className="flex items-center gap-2">
                <Select
                  items={zmPeriods.map((p) => ({ value: p.key, label: p.label }))}
                  value={zmKey}
                  onValueChange={(next) => setZmKey(String(next))}
                  className="w-48"
                />
                <Button
                  variant="secondary"
                  size="sm"
                  loading={busy === 'zm-save'}
                  disabled={!zmReturn || busy !== null || writeLock.locked}
                  title={writeLock.hint ?? (!zmReturn ? zmSheetHint : undefined)}
                  onClick={() => void handleSaveZMDraft()}
                >
                  Entwurf speichern
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  loading={busy === `zm-export-${zmKey}`}
                  disabled={!zmReturn || busy !== null}
                  title={!zmReturn ? zmSheetHint : undefined}
                  onClick={() => void handleExportZM(zmKey)}
                >
                  Als Datei
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={!zmReturn || busy !== null || zmConfirmHint !== undefined || writeLock.locked}
                  title={writeLock.hint ?? (!zmReturn ? zmSheetHint : zmConfirmHint)}
                  onClick={() =>
                    openConfirm({
                      kind: 'zm',
                      periodKey: zmKey,
                      label: `Zusammenfassende Meldung ${selectedZmPeriod?.label ?? zmKey}`,
                    })
                  }
                >
                  Als übermittelt bestätigen
                </Button>
              </div>
            }
          >
            {(zmReturn?.findings ?? []).map((finding) => (
              <Notice key={finding} className="mb-4" tone="attention" text={finding} />
            ))}

            {zmReturn?.reconciliation && (
              <div className="mb-6">
                <StatRow>
                  <Stat
                    label="Innergemeinschaftliche Lieferungen"
                    value={formatCents(zmReturn.reconciliation.suppliesZm)}
                    context={`Kennziffer 41: ${formatCents(zmReturn.reconciliation.suppliesVat)}`}
                    tone={
                      zmReturn.reconciliation.suppliesZm === zmReturn.reconciliation.suppliesVat
                        ? 'neutral'
                        : 'negative'
                    }
                  />
                  <Stat
                    label="Sonstige Leistungen"
                    value={formatCents(zmReturn.reconciliation.servicesZm)}
                    context={`Kennziffer 21: ${formatCents(zmReturn.reconciliation.servicesVat)}`}
                    tone={
                      zmReturn.reconciliation.servicesZm === zmReturn.reconciliation.servicesVat
                        ? 'neutral'
                        : 'negative'
                    }
                  />
                  <Stat
                    label="Abgestimmt gegen"
                    value={zmReturn.reconciliation.scopeLabel || '—'}
                    context={`${zmReturn.reconciliation.vatReturnsFound} Voranmeldungen`}
                  />
                </StatRow>
              </div>
            )}

            {!zmReturn || zmLines.length === 0 ? (
              <EmptyState
                title="Keine meldepflichtigen Umsätze"
                description="Gemeldet werden innergemeinschaftliche Lieferungen und sonstige Leistungen an Abnehmer mit USt-IdNr."
              />
            ) : (
              <Table>
                <Thead>
                  <Tr>
                    <Th className="w-16">Land</Th>
                    <Th className="w-44">USt-IdNr.</Th>
                    <Th>Abnehmer</Th>
                    <Th className="w-40">Art</Th>
                    <Th numeric className="w-40">
                      Betrag
                    </Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {zmLines.map((line) => (
                    <Tr key={`${line.vatId}-${line.kind}`}>
                      <Td code>{line.countryCode}</Td>
                      <Td code>{line.vatId}</Td>
                      <Td>{line.contactName || '—'}</Td>
                      <Td className="text-ink-muted">{ZM_KIND_LABELS[line.kind]}</Td>
                      <Td numeric>{formatCents(line.amount)}</Td>
                    </Tr>
                  ))}
                  <Tr variant="sum">
                    <Td colSpan={4}>Summe</Td>
                    <Td numeric>
                      {formatCents(zmReturn.totalSupplies + zmReturn.totalServices)}
                    </Td>
                  </Tr>
                </Tbody>
              </Table>
            )}
          </Section>

          {zmLateEntries.length > 0 && (
            <Section
              title="Nachträge zu übermittelten Meldezeiträumen"
              context={`${zmLateEntries.length} meldepflichtige Umsätze in bereits übermittelten Zeiträumen`}
              action={
                <HelpPopover label="Erklärung zu den Nachträgen der Meldung">
                  Diese Umsätze gehören in einen Meldezeitraum, dessen Zusammenfassende Meldung
                  bereits übermittelt ist. Nachgemeldet wird nicht im laufenden Zeitraum, sondern
                  über die berichtigte Meldung des ursprünglichen — § 18a Abs. 10 UStG verlangt sie
                  binnen eines Monats nach Erkennen des Fehlers.
                </HelpPopover>
              }
            >
              <Table>
                <Thead>
                  <Tr>
                    <Th className="w-32">Buchung</Th>
                    <Th className="w-28">Datum</Th>
                    <Th className="w-44">USt-IdNr.</Th>
                    <Th>Art</Th>
                    <Th className="w-28">Zeitraum</Th>
                    <Th numeric className="w-32">
                      Betrag
                    </Th>
                    <Th className="w-48" aria-label="Aktion" />
                  </Tr>
                </Thead>
                <Tbody>
                  {zmLateEntries.map((late) => (
                    <Tr key={`${late.entryId}-${late.kind}`}>
                      <Td code>
                        <EntryLink entryNumber={late.entryNumber} onNavigate={onNavigate} />
                      </Td>
                      <Td className="num text-ink-subtle">{formatDate(late.date)}</Td>
                      <Td code>{late.vatId || '—'}</Td>
                      <Td className="text-ink-muted whitespace-normal">
                        {ZM_KIND_LABELS[late.kind]}
                      </Td>
                      <Td className="code-num text-caption text-ink-muted">{late.periodKey}</Td>
                      <Td numeric>{formatCents(late.amount)}</Td>
                      <Td className="text-right">
                        <Button
                          variant="secondary"
                          size="sm"
                          loading={busy === `zm-correction-${late.periodKey}`}
                          disabled={busy !== null || writeLock.locked}
                          title={writeLock.hint}
                          onClick={() => void handleZMCorrection(late.periodKey)}
                        >
                          Berichtigung erzeugen
                        </Button>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Section>
          )}

          <Section
            title="Gespeicherte Meldungen"
            context={`Geschäftsjahr ${year} · Entwürfe und Übermittlungsprotokoll`}
          >
            {savedZM.length === 0 ? (
              <EmptyState
                title="Noch keine Meldung gespeichert"
                description="Ein Entwurf entsteht über den Knopf oben; bestätigt wird er nach dem Hochladen im BZSt-Online-Portal."
              />
            ) : (
              <Table>
                <Thead>
                  <Tr>
                    <Th className="w-32">Zeitraum</Th>
                    <Th className="w-28">Fällig</Th>
                    <Th numeric className="w-36">
                      Summe
                    </Th>
                    <Th className="w-40">Stand</Th>
                    <Th>Transferticket</Th>
                    <Th className="w-28" aria-label="Aktion" />
                  </Tr>
                </Thead>
                <Tbody>
                  {[...savedZM]
                    .sort((a, b) => b.periodKey.localeCompare(a.periodKey) || b.id - a.id)
                    .map((r) => (
                      <Tr key={r.id}>
                        <Td>
                          <span className="flex items-center gap-2">
                            <span className="code-num text-caption text-ink-muted">{r.periodKey}</span>
                            {r.isCorrection && (
                              <span className="text-caption text-attention-text">Berichtigung</span>
                            )}
                          </span>
                        </Td>
                        <Td className="num text-ink-subtle">{formatDate(r.dueDate ?? '')}</Td>
                        <Td numeric>{formatCents(r.totalSupplies + r.totalServices)}</Td>
                        <Td className={r.status === 'submitted' ? 'text-ink' : 'text-ink-subtle'}>
                          {r.status === 'submitted' ? 'Übermittelt' : 'Entwurf'}
                        </Td>
                        <Td className="code-num text-caption text-ink-muted">
                          {r.transferTicket
                            ? `${r.transferTicket} · ${formatDate(r.submittedAt ?? '')}`
                            : '—'}
                        </Td>
                        <Td className="text-right">
                          <Button
                            variant="quiet"
                            size="sm"
                            loading={busy === `zm-export-${r.id}`}
                            disabled={busy !== null}
                            onClick={() => void handleExportZM(r.periodKey, r.id)}
                          >
                            Als Datei
                          </Button>
                        </Td>
                      </Tr>
                    ))}
                </Tbody>
              </Table>
            )}
          </Section>
        </>
      )}

      <Dialog
        open={confirming !== null}
        onOpenChange={(next) => !next && setConfirming(null)}
        title={confirming ? `${confirming.label} bestätigen` : ''}
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirming(null)}>
              Abbrechen
            </Button>
            <Button
              variant="primary"
              loading={busy === 'confirm'}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void handleConfirm()}
            >
              Übermittlung bestätigen
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-4">
          <Field label="Datum der Übermittlung">
            <Input
              type="date"
              value={confirmDate}
              onChange={(e) => setConfirmDate(e.target.value)}
            />
          </Field>
          <Field
            label="Transferticket"
            error={ticketError ?? undefined}
            help="Es ist der Nachweis, dass die Anmeldung angekommen ist."
          >
            <Input
              className="code-num"
              value={confirmTicket}
              onChange={(e) => {
                setConfirmTicket(e.target.value);
                if (ticketError) setTicketError(null);
              }}
              placeholder="aus Mein ELSTER bzw. dem BZSt-Online-Portal"
            />
          </Field>
          <Field label="Notiz" optional>
            <Textarea value={confirmNote} onChange={(e) => setConfirmNote(e.target.value)} />
          </Field>
          {/* Der abgelehnte Versuch bleibt über den Aktionen stehen, bis er
              behoben ist — er gehört an den Dialog und nicht in einen Toast. */}
          {confirmError && <Notice tone="negative" text={confirmError} />}
        </div>
      </Dialog>
    </div>
  );
};

// -------------------------------------------------------------------------

const ZM_KIND_LABELS: Record<string, string> = {
  L: 'Innergemeinschaftliche Lieferung',
  S: 'Sonstige Leistung § 3a Abs. 2',
  D: 'Dreiecksgeschäft',
};

/**
 * Eine Kennziffernzeile mit ihrem Drill-down.
 *
 * Ohne die Buchungen dahinter ist eine Kennziffer eine Zahl, die niemand prüfen
 * kann (UST-01). Aufgeklappt wird deshalb an der Zeile selbst, nicht in einer
 * zweiten Ansicht.
 */
const FigureRow: React.FC<{
  line: VatReturnLine;
  expanded: boolean;
  onToggle: () => void;
  entryById: Map<number, JournalEntry>;
  onNavigate?: NavigateFn;
}> = ({ line, expanded, onToggle, entryById, onNavigate }) => {
  const drillable = (line.entryIds?.length ?? 0) > 0;
  const deviation = line.hasTax && line.hasBase ? line.tax - line.expectedTax : 0;
  const empty = line.base === 0 && line.tax === 0;

  return (
    <>
      {/* Die Zahllast ist die Summe des Blatts und trägt die buchhalterische
          Doppellinie — sie ist keine Position unter den anderen. */}
      <Tr variant={line.code === PAYABLE_CODE ? 'sum' : 'default'}>
        <Td code>
          {drillable ? (
            <button
              type="button"
              onClick={onToggle}
              aria-expanded={expanded}
              className="inline-flex items-center gap-1 code-num text-ink-muted
                         hover:text-ink transition-colors duration-120 ease-quiet"
            >
              {expanded ? (
                <ChevronDown className="w-3.5 h-3.5" strokeWidth={1.5} />
              ) : (
                <ChevronRight className="w-3.5 h-3.5" strokeWidth={1.5} />
              )}
              {line.code}
              {line.taxCode && ` / ${line.taxCode}`}
            </button>
          ) : (
            <span className="pl-[18px]">
              {line.code}
              {line.taxCode && ` / ${line.taxCode}`}
            </span>
          )}
        </Td>
        <Td className={cn('whitespace-normal', empty && 'text-ink-subtle')}>{line.label}</Td>
        <Td numeric className="text-ink-muted">
          {line.hasBase ? formatCents(line.base) : '—'}
        </Td>
        <Td numeric>{line.hasTax ? formatCents(line.tax) : '—'}</Td>
        <Td numeric className={deviation !== 0 ? 'text-attention-text' : 'text-ink-subtle'}>
          {deviation !== 0 ? formatCents(deviation) : '—'}
        </Td>
      </Tr>

      {expanded &&
        (line.entryIds ?? []).map((id) => {
          const entry = entryById.get(id);
          return (
            // Die Buchung steht über die vier Spalten hinweg: Nummer, Datum,
            // Text und Betrag sind keine Kennziffernwerte und gehören deshalb
            // nicht unter deren Überschriften.
            <Tr key={`${line.code}-${id}`}>
              <Td />
              <Td colSpan={4} className="text-ink-muted">
                <span className="flex items-center gap-3">
                  <EntryLink entryNumber={entry?.entryNumber ?? `#${id}`} onNavigate={onNavigate} />
                  <span className="num text-ink-subtle">
                    {entry ? formatDate(entry.bookingDate) : '—'}
                  </span>
                  <span className="flex-1 truncate">
                    {entry?.description ?? 'Buchung nicht in diesem Geschäftsjahr'}
                  </span>
                  <span className="num">{entry ? formatCents(debitTotal(entry)) : '—'}</span>
                </span>
              </Td>
            </Tr>
          );
        })}
    </>
  );
};

/** Die Buchungsnummer führt ins Journal, gefiltert auf genau diese Buchung. */
const EntryLink: React.FC<{ entryNumber: string; onNavigate?: NavigateFn }> = ({
  entryNumber,
  onNavigate,
}) =>
  onNavigate ? (
    <button
      type="button"
      onClick={() => onNavigate('journal', { entryNumber })}
      className="code-num text-accent-text hover:text-accent transition-colors duration-120 ease-quiet"
    >
      {entryNumber}
    </button>
  ) : (
    <span className="code-num">{entryNumber}</span>
  );

/** Der Bruttobetrag einer Buchung: die Summe ihrer Sollseite. */
function debitTotal(entry: JournalEntry): number {
  return (entry.lines ?? [])
    .filter((l) => l.side === 'S')
    .reduce((sum, l) => sum + l.amount, 0);
}

/** Der Zeitraum, der heute ansteht — sonst der zuletzt gewählte. */
function pickPeriod<T extends { key: string; from: string; to: string }>(
  periods: T[],
  current: string,
): string {
  if (current && periods.some((p) => p.key === current)) return current;
  const now = today();
  const running = periods.find((p) => p.from <= now && p.to >= now);
  return running?.key ?? periods[periods.length - 1]?.key ?? '';
}

function today(): string {
  return new Date().toISOString().slice(0, 10);
}

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

