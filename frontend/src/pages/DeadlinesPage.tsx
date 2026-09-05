import React, { useEffect, useState } from 'react';
import { Lock } from 'lucide-react';
import { CheckRun, CompanySettings, Deadline, Festschreibung, FoundationState } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatDate } from '../utils/formatters';
import type { NavigateFn } from '../components/Sidebar';
import { FoundationDutyResetDialog, FoundationSection } from '../components/FoundationSection';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Notice,
  PageHeader,
  Section,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tabs,
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
 * Steuerfristen.
 *
 * Die Termine kommen aus `GetDeadlines` und werden hier nicht mehr gerechnet:
 * Voranmeldung, Zusammenfassende Meldung, Sondervorauszahlung, Festschreibung,
 * Aufstellung, Offenlegung und die Gründungspflichten stehen im Backend an
 * einer Stelle, mitsamt ihrer Norm. „Erledigt" ergibt sich dort aus den Daten —
 * eine übermittelte Voranmeldung ist abgegeben, ein festgeschriebener Monat ist
 * festgeschrieben. Der Haken im `localStorage` ist damit weg; er war eine
 * Aussage über den Browser und nicht über den Mandanten.
 *
 * Vor der Festschreibung läuft der Prüfbericht. Was er als blockierend meldet,
 * lässt sich nur mit einer Begründung übergehen, und die steht danach am
 * Prüflauf und im Protokoll (GoBD Rz. 34 ff.).
 */

/** Die Art eines Termins steht als Wort, nicht als Farbe (§3.4). */
const CATEGORY_BY_PREFIX: [string, string][] = [
  ['ustva.', 'Voranmeldung'],
  ['zm.', 'Zusammenfassende Meldung'],
  ['sondervorauszahlung.', 'Sondervorauszahlung'],
  ['festschreibung.', 'Festschreibung'],
  ['ust.jahreserklaerung', 'Jahreserklärung'],
  ['gruendung.', 'Gründung'],
  ['abschluss.', 'Jahresabschluss'],
];

function categoryOf(key: string): string {
  for (const [prefix, label] of CATEGORY_BY_PREFIX) {
    if (key.startsWith(prefix)) return label;
  }
  return 'Termin';
}

/**
 * Von Hand abgehakt wird nur, was Buchfink nicht sieht: die
 * Umsatzsteuer-Jahreserklärung entsteht außerhalb. Alles andere ergibt sich aus
 * den Daten, und ein Haken daneben könnte ihnen nur widersprechen.
 */
function isManual(key: string): boolean {
  return key.startsWith('ust.jahreserklaerung');
}

/** Die Gründungspflichten tragen ihr Erledigungsdatum am Vorgang selbst. */
function dutyKeyOf(key: string): string | null {
  return key.startsWith('gruendung.') ? key.slice('gruendung.'.length) : null;
}

type Filter = 'all' | 'open' | 'overdue' | 'done';

interface CommittablePeriod {
  /**
   * `year` trägt der letzte Zeitraum des Geschäftsjahres: er schreibt zugleich
   * das Jahr fest, und der Prüfbericht davor nimmt die Regeln des Abschlusses
   * hinzu (Abschreibung, Gliederungszuordnung).
   */
  type: 'month' | 'quarter' | 'year';
  label: string;
  cutoff: string; // YYYY-MM-DD, last day of the period
}

/** Der Prüfbericht vor einer Festschreibung, samt Begründung zum Übergehen. */
interface CommitDialogState {
  period: CommittablePeriod;
  run: CheckRun | null;
  reason: string;
}

/**
 * Die Kennung des Begründungsfeldes. Der Fokus springt beim Absenden dorthin
 * (§8.3), und dafür braucht das Feld eine Adresse.
 */
const COMMIT_REASON_FIELD = 'commit-override-reason';

/**
 * Die Folge der Festschreibung, in einem Satz. Sie steht im Dialog unabhängig
 * davon, ob der Prüfbericht etwas gefunden hat: § 8.2 des Design-Konzepts
 * verlangt, dass der Dialog die Folge benennt und nicht die Aktion — und
 * unumkehrbar ist der Schritt auch dann, wenn nichts zu beanstanden war.
 */
const COMMIT_CONSEQUENCE =
  'Festgeschriebene Buchungen lassen sich nur noch per Storno korrigieren.';

/**
 * Was der letzte Zeitraum des Jahres zusätzlich bedeutet. Er schreibt das
 * Geschäftsjahr mit fest, und deshalb prüft der Bericht davor auch die
 * Abschreibung und die Gliederungszuordnung. Ohne diesen Satz wäre der
 * strengere Prüflauf eine Überraschung.
 */
const YEAR_CONSEQUENCE =
  'Mit diesem Zeitraum wird zugleich das Geschäftsjahr festgeschrieben; der Prüfbericht ' +
  'nimmt deshalb die Abschreibung und die Zuordnung der Konten zur Gliederung hinzu.';

export interface DeadlinesPageProps {
  /** Weg vom Befund zu der Stelle, an der er zu beheben ist. */
  onNavigate?: NavigateFn;
}

export const DeadlinesPage: React.FC<DeadlinesPageProps> = ({ onNavigate }) => {
  // Festschreiben und Erledigtvermerke schreiben; die Fristenliste selbst ist
  // eine Auswertung und bleibt im Prüfermodus lesbar (§10.4).
  const writeLock = useWriteLock();
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [deadlines, setDeadlines] = useState<Deadline[]>([]);
  const [activeFilter, setActiveFilter] = useState<Filter>('all');
  const [festschreibungen, setFestschreibungen] = useState<Festschreibung[]>([]);
  const [foundation, setFoundation] = useState<FoundationState | null>(null);
  const [resettingDuty, setResettingDuty] = useState<Deadline | null>(null);
  const [commitDialog, setCommitDialog] = useState<CommitDialogState | null>(null);
  const [checking, setChecking] = useState<string | null>(null);
  const [committing, setCommitting] = useState(false);
  // Die fehlende Begründung wird am Feld gemeldet und nicht durch einen
  // gesperrten Knopf verschwiegen (§8.3).
  const [reasonError, setReasonError] = useState<string | null>(null);
  const [markingKey, setMarkingKey] = useState<string | null>(null);

  const currentYear = settings?.fiscalYear || new Date().getFullYear();

  useEffect(() => {
    void loadAll();
  }, []);

  const loadAll = async () => {
    try {
      const cfg = await Api.getCompanySettings();
      setSettings(cfg);
      const year = cfg.fiscalYear || new Date().getFullYear();
      setDeadlines(await Api.getDeadlines(year));
      setFestschreibungen(await Api.getFestschreibungen());
      setFoundation(await Api.getFoundationState());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  /**
   * Eine Gründungspflicht wird mit ihrem Datum quittiert, nicht mit einem Haken.
   * Zurückgenommen wird sie über die Rückfrage — ein Klick, der ein Datum
   * löscht, gehört nicht in eine Tabellenzeile.
   */
  const toggleDuty = async (item: Deadline) => {
    const dutyKey = dutyKeyOf(item.key);
    if (!dutyKey) return;
    if (item.isDone) {
      setResettingDuty(item);
      return;
    }
    setMarkingKey(item.key);
    try {
      await Api.completeFoundationDuty(dutyKey, new Date().toISOString().slice(0, 10));
      await loadAll();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setMarkingKey(null);
    }
  };

  const resetDuty = async (item: Deadline) => {
    const dutyKey = dutyKeyOf(item.key);
    if (!dutyKey) return;
    try {
      await Api.completeFoundationDuty(dutyKey, '');
      await loadAll();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  /** Der eine Termin, den Buchfink nicht aus den Daten ablesen kann. */
  const markManual = async (item: Deadline) => {
    setMarkingKey(item.key);
    try {
      await Api.markDeadlineDone(item.key, new Date().toISOString().slice(0, 10));
      await loadAll();
      toast.success(`${item.title} als erledigt vermerkt.`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setMarkingKey(null);
    }
  };

  // The committable accounting periods of the fiscal year, derived from the VAT
  // reporting cadence, limited to periods that have already ended.
  const getCommittablePeriods = (): CommittablePeriod[] => {
    const y = currentYear;
    const todayIso = new Date().toISOString().slice(0, 10);
    const monthNames = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];
    const lastDay = (year: number, month1: number) => new Date(year, month1, 0).getDate(); // month1 is 1-based
    const periods: CommittablePeriod[] = [];

    if ((settings?.vatPeriod || 'quarter') === 'month') {
      for (let m = 1; m <= 12; m++) {
        const cutoff = `${y}-${String(m).padStart(2, '0')}-${String(lastDay(y, m)).padStart(2, '0')}`;
        periods.push({ type: 'month', label: `${monthNames[m - 1]} ${y}`, cutoff });
      }
    } else {
      const quarters: [string, number][] = [['Q1', 3], ['Q2', 6], ['Q3', 9], ['Q4', 12]];
      for (const [label, endMonth] of quarters) {
        const cutoff = `${y}-${String(endMonth).padStart(2, '0')}-${String(lastDay(y, endMonth)).padStart(2, '0')}`;
        periods.push({ type: 'quarter', label: `${label} ${y}`, cutoff });
      }
    }
    // Der letzte Zeitraum des Jahres ist zugleich die Jahresfestschreibung:
    // nach ihm nimmt das Geschäftsjahr keine Buchung mehr auf. Er läuft deshalb
    // als Zeitraumtyp „year" — nur dann prüft der Bericht davor die
    // Abschreibung und die Gliederungszuordnung, die zum Abschluss gehören und
    // nicht zum Monat. Ein eigener Eintrag daneben ginge nicht: er trüge
    // denselben Stichtag, und zweimal derselbe Stichtag ist keine zweite
    // Festschreibung.
    const last = periods[periods.length - 1];
    if (last) {
      last.type = 'year';
      last.label = `${last.label} · Jahresabschluss`;
    }
    return periods.filter((p) => p.cutoff <= todayIso);
  };

  const committedCutoffs = new Set(festschreibungen.map((f) => f.cutoffDate));
  const latestCommittedCutoff = festschreibungen.reduce((max, f) => (f.cutoffDate > max ? f.cutoffDate : max), '');

  /**
   * Der Prüfbericht kommt vor die Festschreibung, nicht danach: er sagt, was
   * hinterher nicht mehr zu ändern wäre.
   */
  const openCommitDialog = async (p: CommittablePeriod) => {
    setChecking(p.cutoff);
    setReasonError(null);
    try {
      const run = await Api.runChecks(p.cutoff, p.type);
      setCommitDialog({ period: p, run, reason: '' });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setChecking(null);
    }
  };

  const handleCommitPeriod = async () => {
    if (!commitDialog) return;
    const { period, reason } = commitDialog;
    // Der Absenden-Knopf bleibt aktiv; fehlt die Begründung, steht der Fehler am
    // Feld und der Fokus springt dorthin (§8.3). Ein gesperrter Knopf verschwiege
    // den Grund.
    if (blocking.length > 0 && reason.trim() === '') {
      setReasonError('Blockierende Befunde lassen sich nur mit einer Begründung übergehen.');
      document.getElementById(COMMIT_REASON_FIELD)?.focus();
      return;
    }
    setReasonError(null);
    setCommitting(true);
    try {
      const rec = await Api.commitPeriod(period.type, period.label, period.cutoff, reason.trim());
      if (rec) {
        toast.success(
          rec.timestampStatus === 'confirmed'
            ? `${period.label} festgeschrieben, beglaubigt durch ${rec.tsaName}.`
            : `${period.label} festgeschrieben. Der Zeitstempel wird nachgeholt, sobald wieder Netz da ist.`,
        );
        setCommitDialog(null);
        await loadAll();
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setCommitting(false);
    }
  };

  const today = new Date();
  today.setHours(0, 0, 0, 0);

  /** Tage bis zur Fälligkeit. Negativ heißt überfällig. */
  const daysUntil = (iso: string) => {
    const due = new Date(iso);
    due.setHours(0, 0, 0, 0);
    return Math.ceil((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
  };

  const statusOf = (item: Deadline) => {
    if (item.isDone) return 'done';
    // Ohne Tagesfrist gibt es kein Überfällig — „unverzüglich" lässt sich nicht
    // in Tagen messen.
    if (!item.dueDate) return 'open';
    const diff = daysUntil(item.dueDate);
    if (diff < 0) return 'overdue';
    if (diff <= 30) return 'upcoming';
    return 'open';
  };

  const doneCount = deadlines.filter((d) => d.isDone).length;
  const openCount = deadlines.length - doneCount;
  const overdueCount = deadlines.filter((d) => statusOf(d) === 'overdue').length;
  const upcomingCount = deadlines.filter((d) => statusOf(d) === 'upcoming').length;

  const filtered = deadlines.filter((item) => {
    const status = statusOf(item);
    if (activeFilter === 'open' && status === 'done') return false;
    if (activeFilter === 'overdue' && status !== 'overdue') return false;
    if (activeFilter === 'done' && status !== 'done') return false;
    return true;
  });

  const periods = getCommittablePeriods();
  // Ein Prüflauf ohne Befund ist der gute Fall. Go serialisiert die leere Liste
  // als `null`, und `null.length` im Dialog nähme die Ansicht mit — genau dann,
  // wenn nichts zu beanstanden ist. Die Liste wird deshalb einmal auf einen
  // Wert gebracht und danach nur noch von hier gelesen.
  const findings = commitDialog?.run?.findings ?? [];
  const blocking = findings.filter((f) => f.severity === 'blocking');
  const warnings = findings.filter((f) => f.severity === 'warning');

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title={`Steuerfristen ${currentYear}`}
        context="Voranmeldungen, Meldungen und Festschreibung"
        action={
          <HelpPopover label="Erklärung zu den Steuerfristen">
            Die Termine und ihr Stand kommen aus den Daten: eine übermittelte Voranmeldung ist
            abgegeben, ein festgeschriebener Monat ist festgeschrieben. Bei Überweisung an das
            Finanzamt gilt die Zahlungsschonfrist von drei Tagen nach § 240 Abs. 3 AO; fällt ein
            Fälligkeitstag auf ein Wochenende, verschiebt er sich auf den nächsten Werktag.
          </HelpPopover>
        }
      />

      <div className="mt-6">
        <StatRow>
          <Stat label="Termine" value={String(deadlines.length)} context={`im Jahr ${currentYear}`} />
          <Stat
            label="Überfällig"
            value={String(overdueCount)}
            context={overdueCount > 0 ? 'Frist verstrichen' : 'keine Rückstände'}
            tone={overdueCount > 0 ? 'negative' : 'neutral'}
          />
          <Stat
            label="Demnächst fällig"
            value={String(upcomingCount)}
            context="in den nächsten 30 Tagen"
          />
          <Stat label="Erledigt" value={String(doneCount)} context={`${openCount} noch offen`} />
        </StatRow>
      </div>

      {foundation?.applies && foundation.hasFoundation && foundation.stage === 'vorgesellschaft' && (
        <FoundationSection state={foundation} onChanged={loadAll} />
      )}

      <Section
        title="Zeiträume festschreiben"
        context="Abschluss passend zum Rhythmus der Voranmeldung"
        action={
          <HelpPopover label="Erklärung zur Festschreibung">
            Vor der Festschreibung läuft der Prüfbericht: er nennt, was danach nicht mehr zu ändern
            wäre. Ein festgeschriebener Zeitraum nimmt keine rückdatierten Buchungen mehr an,
            Korrekturen laufen ab dann über den Storno. Zusätzlich beglaubigt ein unabhängiger
            Zeitstempeldienst den Stand — übertragen wird dabei nur eine Prüfsumme.
          </HelpPopover>
        }
      >
        {periods.length === 0 ? (
          <EmptyState
            icon={<Lock className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch kein abgeschlossener Zeitraum"
            description="Zeiträume lassen sich festschreiben, sobald sie beendet sind."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Zeitraum</Th>
                <Th className="w-40">Stichtag</Th>
                <Th className="w-56" aria-label="Zustand" />
              </Tr>
            </Thead>
            <Tbody>
              {periods.map((period) => {
                const committed = committedCutoffs.has(period.cutoff);
                const busy = checking === period.cutoff;
                // Festgeschrieben wird der Reihe nach. Ein Zeitraum mit Lücke
                // davor wäre kein Abschluss.
                const isNext = !committed && period.cutoff > latestCommittedCutoff;
                return (
                  <Tr key={period.cutoff}>
                    <Td>{period.label}</Td>
                    <Td className="num text-ink-subtle">{formatDate(period.cutoff)}</Td>
                    <Td className="text-right">
                      {committed ? (
                        <StatusBadge status="festgeschrieben" />
                      ) : (
                        <Button
                          variant="secondary"
                          size="sm"
                          loading={busy}
                          disabled={!isNext || checking !== null || writeLock.locked}
                          title={
                            writeLock.hint ??
                            (!isNext ? 'Frühere Zeiträume zuerst festschreiben' : undefined)
                          }
                          onClick={() => void openCommitDialog(period)}
                          icon={<Lock className="w-3.5 h-3.5" strokeWidth={1.5} />}
                        >
                          Prüfen und festschreiben
                        </Button>
                      )}
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section title="Termine" context="Stand und Frist folgen den Daten">
        <Tabs
          items={[
            { value: 'all' as Filter, label: 'Alle', count: deadlines.length },
            { value: 'open' as Filter, label: 'Offen', count: openCount },
            { value: 'overdue' as Filter, label: 'Überfällig', count: overdueCount },
            { value: 'done' as Filter, label: 'Erledigt', count: doneCount },
          ]}
          value={activeFilter}
          onValueChange={setActiveFilter}
          className="mb-6"
        />

        {filtered.length === 0 ? (
          <EmptyState
            variant={deadlines.length === 0 ? 'leer' : 'gefiltert'}
            title={
              deadlines.length === 0 ? 'Keine Termine für dieses Jahr' : 'Kein Termin in dieser Auswahl'
            }
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Termin</Th>
                <Th className="w-48">Art</Th>
                <Th className="w-32">Fällig am</Th>
                <Th className="w-52">Zustand</Th>
                <Th className="w-44" aria-label="Aktion" />
              </Tr>
            </Thead>
            <Tbody>
              {filtered.map((item) => {
                const diff = item.dueDate ? daysUntil(item.dueDate) : null;
                const dutyKey = dutyKeyOf(item.key);
                return (
                  <Tr key={item.key}>
                    <Td className="max-w-[30rem]">
                      <span className="flex items-center gap-1">
                        <span className={cn('truncate', item.isDone && 'text-ink-subtle')}>
                          {item.title}
                        </span>
                        {/* Beschreibung und Norm sind zwei Sätze — das ist ein
                            Popover und kein Tooltip (§15.2). */}
                        <HelpPopover label={`Erklärung zu ${item.title}`}>
                          {`${item.description} Frist: ${item.period} (${item.reference}).`}
                        </HelpPopover>
                      </span>
                    </Td>
                    <Td className="text-ink-muted">{categoryOf(item.key)}</Td>
                    <Td className={cn(item.dueDate ? 'num text-ink-subtle' : 'text-ink-subtle')}>
                      {item.dueDate ? formatDate(item.dueDate) : (item.period || '—')}
                    </Td>
                    <Td>
                      {item.isDone ? (
                        <span className="text-caption text-ink-subtle">
                          {item.doneOn ? `Erledigt am ${formatDate(item.doneOn)}` : 'Erledigt'}
                        </span>
                      ) : diff === null ? (
                        <span className="text-caption text-ink-subtle">Offen</span>
                      ) : diff < 0 ? (
                        <span className="flex items-center gap-2">
                          <StatusBadge status="ueberfaellig" />
                          <span className="text-caption text-ink-subtle num">
                            seit {Math.abs(diff)} Tagen
                          </span>
                        </span>
                      ) : diff === 0 ? (
                        <span className="text-caption text-attention-text">Heute fällig</span>
                      ) : diff <= 30 ? (
                        <span className="text-caption text-attention-text num">in {diff} Tagen</span>
                      ) : (
                        <span className="text-caption text-ink-subtle">Offen</span>
                      )}
                    </Td>
                    <Td className="text-right">
                      {dutyKey ? (
                        <Button
                          variant="quiet"
                          size="sm"
                          loading={markingKey === item.key}
                          disabled={writeLock.locked}
                          title={writeLock.hint}
                          onClick={() => void toggleDuty(item)}
                        >
                          {item.isDone ? 'Als offen führen' : 'Erledigt vermerken'}
                        </Button>
                      ) : isManual(item.key) && !item.isDone ? (
                        <Button
                          variant="quiet"
                          size="sm"
                          loading={markingKey === item.key}
                          disabled={writeLock.locked}
                          title={writeLock.hint}
                          onClick={() => void markManual(item)}
                        >
                          Erledigt vermerken
                        </Button>
                      ) : null}
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <FoundationDutyResetDialog
        title={resettingDuty?.title ?? null}
        onOpenChange={(next) => !next && setResettingDuty(null)}
        onConfirm={() => {
          const item = resettingDuty;
          setResettingDuty(null);
          if (item) void resetDuty(item);
        }}
      />

      <Dialog
        open={commitDialog !== null}
        onOpenChange={(next) => !next && setCommitDialog(null)}
        title={commitDialog ? `${commitDialog.period.label} festschreiben` : ''}
        width="max-w-3xl"
        footer={
          <>
            <Button variant="secondary" onClick={() => setCommitDialog(null)}>
              Abbrechen
            </Button>
            <Button
              variant="primary"
              loading={committing}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void handleCommitPeriod()}
            >
              Festschreiben
            </Button>
          </>
        }
      >
        {commitDialog?.run && (
          <div className="flex flex-col gap-5">
            <StatRow>
              <Stat
                label="Geprüfte Buchungen"
                value={String(commitDialog.run.checkedEntries)}
                context={`bis ${formatDate(commitDialog.period.cutoff)}`}
              />
              <Stat label="Geprüfte Belege" value={String(commitDialog.run.checkedReceipts)} />
              <Stat label="Geprüfte Bankumsätze" value={String(commitDialog.run.checkedBankTx)} />
              <Stat
                label="Befunde"
                value={`${blocking.length} · ${warnings.length}`}
                context="blockierend · Hinweise"
                tone={blocking.length > 0 ? 'negative' : 'neutral'}
              />
            </StatRow>

            {/* Der Dialog benennt die Folge, nicht die Aktion (§8.2) — auch
                dann, wenn der Prüfbericht nichts gefunden hat. */}
            <p className="text-body text-ink-muted">
              {COMMIT_CONSEQUENCE}
              {commitDialog.period.type === 'year' && ` ${YEAR_CONSEQUENCE}`}
            </p>

            {blocking.length > 0 && (
              <Notice
                tone="negative"
                text="Festgeschrieben wird ein Stand, der danach nicht mehr zu ändern ist — diese Befunde bleiben darin stehen."
              />
            )}

            {findings.length === 0 ? (
              <p className="text-body text-ink-muted">Der Prüfbericht hat nichts gefunden.</p>
            ) : (
              // Die Befunde stehen als Zeilen, durch Haarlinien getrennt: eine
              // Tabelle wäre hier eine Fläche im Dialog und damit eine Fläche in
              // einer Fläche (§6.3).
              <ul className="flex flex-col">
                {[...blocking, ...warnings].map((finding) => (
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
                        onClick={() => {
                          const target = findingTarget(finding.objectType);
                          if (!target) return;
                          setCommitDialog(null);
                          onNavigate(
                            target,
                            finding.objectType === 'JOURNAL_ENTRY'
                              ? { entryNumber: finding.objectName }
                              : finding.objectType === 'ACCOUNT'
                                ? { account: finding.objectId }
                                : {},
                          );
                        }}
                      >
                        Hin dazu
                      </Button>
                    )}
                  </li>
                ))}
              </ul>
            )}

            {blocking.length > 0 && (
              <Field
                label="Begründung für das Übergehen"
                hint="steht am Prüflauf und im Protokoll"
                error={reasonError ?? undefined}
              >
                <Textarea
                  id={COMMIT_REASON_FIELD}
                  value={commitDialog.reason}
                  onChange={(e) => {
                    setCommitDialog({ ...commitDialog, reason: e.target.value });
                    if (reasonError) setReasonError(null);
                  }}
                />
              </Field>
            )}
          </div>
        )}
      </Dialog>
    </div>
  );
};

/** Wohin ein Befund führt. Ohne Adresse ist er eine Hausaufgabe ohne Ort. */
function findingTarget(objectType?: string): 'journal' | 'receipts' | 'bank' | 'accounts' | 'vat' | null {
  switch (objectType) {
    case 'JOURNAL_ENTRY':
      return 'journal';
    case 'RECEIPT':
      return 'receipts';
    case 'BANK_TX':
      return 'bank';
    case 'ACCOUNT':
      return 'accounts';
    case 'VAT_PERIOD':
      return 'vat';
    default:
      return null;
  }
}
