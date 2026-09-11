import React, { useCallback, useEffect, useState } from 'react';
import { FileText, Landmark } from 'lucide-react';
import { Api } from '../services/api';
import { MonthCloseDialog } from '../components/MonthCloseDialog';
import { GruendungHelpDialog, GruendungHelpMark } from '../components/GruendungHelp';
import { formatCents, formatDate } from '../utils/formatters';
import { monthOptions, previousMonth } from '../utils/months';
import { targetLabel } from '../utils/findings';
import type {
  FinancialSummary,
  FoundationState,
  JournalEntry,
  MonthCloseState,
  Task,
  TaskList,
} from '../types';
import type { NavigateFn, NavigationParams, TabType } from '../components/Sidebar';
import {
  Button,
  EmptyState,
  Help,
  Notice,
  PageHeader,
  Section,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Die Aufgabenliste als Startseite (Architektur 6.1).
 *
 * Sie rechnet nichts nach: Gruppen, Reihenfolge, Titel und Begründung kommen aus
 * `GetTasks`. Wer keine Buchhalterin ist, weiß nach dem Start nicht, was heute
 * dran ist — und ein Bankguthaben beantwortet das nicht. Die Kennzahlen stehen
 * deshalb darunter und nicht darüber.
 *
 * Die frühere Seite „Übersicht" ist in diese aufgegangen (Architektur 6.1): sie
 * beantwortete dieselbe Frage ein zweites Mal und stand als zweiter Eintrag in
 * derselben Navigationsgruppe. Ihr Kennzahlenblock und die zuletzt erfassten
 * Vorgänge stehen jetzt hier unter der Liste — eine Startseite, nicht zwei.
 */

interface TasksPageProps {
  onNavigate: NavigateFn;
}

/** Eine der drei Gruppen mit ihrer Überschrift und der Farbe ihrer Marke. */
interface TaskGroup {
  key: keyof Pick<TaskList, 'overdue' | 'open' | 'upcoming'>;
  title: string;
  context: string;
  mark: string;
}

/** Die drei Gruppen in der Reihenfolge, in der sie zu lesen sind. */
const GROUPS: TaskGroup[] = [
  {
    key: 'overdue',
    title: 'Überfällig',
    context: 'Fristen, die verstrichen sind',
    mark: 'bg-negative',
  },
  { key: 'open', title: 'Offen', context: 'Arbeit ohne Frist', mark: 'bg-attention' },
  {
    key: 'upcoming',
    title: 'Demnächst',
    context: 'Fristen der nächsten dreißig Tage',
    mark: 'bg-accent',
  },
];

/**
 * Die Seiten, die das Backend als Ziel nennt, sind die Reiter der Navigation.
 * Ein unbekannter Name führt nirgendwohin — dann steht die Aufgabe ohne Knopf
 * da, statt auf einer beliebigen Seite zu landen.
 */
const TARGET_TABS: TabType[] = [
  'accounts',
  'assets',
  'audit',
  'advances',
  'backup',
  'bank',
  'closing',
  'closingmodules',
  'contacts',
  'deadlines',
  'ebilanz',
  'invoices',
  'journal',
  'obligations',
  'receipts',
  'reports',
  'settings',
  'taxaudit',
  'vat',
];

function targetTab(page: string): TabType | null {
  return (TARGET_TABS as string[]).includes(page) ? (page as TabType) : null;
}

/**
 * Übersetzt die Parameter des Ziels in die der Navigation.
 *
 * Das Backend kennt den Router nicht und liefert Zeichenketten; welche davon
 * eine Ansicht versteht, steht hier. Ein Parameter, den keine Ansicht kennt,
 * bleibt weg: er würde stumm ignoriert und wäre dann nur ein Versprechen.
 *
 * Das Jahr ist der wichtigste davon: Abschluss und Abschlussbausteine folgen dem
 * Geschäftsjahr aus der Kopfzeile, und eine Aufgabe zum Vorjahresabschluss führte
 * ohne es auf die gleich aussehende Seite des laufenden Jahres.
 */
function targetParams(task: Task): NavigationParams {
  const params = task.target?.params ?? {};
  const next: NavigationParams = {};
  if (params.account) next.account = params.account;
  if (params.entryNumber) next.entryNumber = params.entryNumber;
  if (params.status) next.receiptStatus = params.status;
  // Der Beleg, den die Aufgabe meint. Ohne ihn führte die Aufgabe
  // „Leistungsnachweis erfassen" auf eine Liste, in der der gebuchte Beleg
  // unter keinem Filter steht.
  const receiptId = Number.parseInt(params.receiptId ?? '', 10);
  if (Number.isFinite(receiptId) && receiptId > 0) next.receiptId = receiptId;
  if (params.view) next.bankView = params.view;
  if (params.rule) next.auditRule = params.rule;
  if (params.key) next.deadlineKey = params.key;
  const year = Number.parseInt(params.year ?? '', 10);
  if (Number.isFinite(year) && year > 0) next.year = year;
  return next;
}

/**
 * Der Kontext einer Zeile: Anzahl, Betrag und Fälligkeit, soweit sie etwas
 * aussagen. Eine Null ist keine Auskunft und steht deshalb nicht da.
 */
function taskContext(task: Task): string {
  const parts: string[] = [];
  if (task.count > 0) parts.push(task.count === 1 ? '1 Vorgang' : `${task.count} Vorgänge`);
  if (task.amount !== 0) parts.push(formatCents(task.amount));
  if (task.dueDate) parts.push(`fällig ${formatDate(task.dueDate)}`);
  return parts.join(' · ');
}

export const TasksPage: React.FC<TasksPageProps> = ({ onNavigate }) => {
  const [tasks, setTasks] = useState<TaskList | null>(null);
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [recentEntries, setRecentEntries] = useState<JournalEntry[]>([]);
  const [monthState, setMonthState] = useState<MonthCloseState | null>(null);
  const [month, setMonth] = useState(() => previousMonth(new Date()));
  // Warum es für den Monat keinen Stand gibt, sagt das Backend. Der Satz bleibt
  // stehen, bis ein anderer Monat gewählt ist (§10.4).
  const [monthError, setMonthError] = useState('');
  const [monthOpen, setMonthOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [foundation, setFoundation] = useState<FoundationState | null>(null);
  const [gruendungHelp, setGruendungHelp] = useState(false);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      setTasks(await Api.getTasks());
    } catch (e) {
      setTasks(null);
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadTasks();
  }, [loadTasks]);

  useEffect(() => {
    // Die Kennzahlen sind Nebenauskunft: fehlen sie, bleibt die Aufgabenliste
    // die Aufgabenliste.
    Api.getFinancialSummary()
      .then(setSummary)
      .catch(() => setSummary(null));
    // Ebenso die zuletzt erfassten Vorgänge. Die Liste kommt ungefiltert aus
    // dem Backend; genommen werden die letzten acht, das jüngste zuerst.
    Api.getJournalEntries()
      .then((entries) => setRecentEntries((entries ?? []).slice(-8).reverse()))
      .catch(() => setRecentEntries([]));
    // Der Gründungsstand ebenso. Er entscheidet über den Hinweisstreifen, und
    // ohne ihn steht die Seite da wie zuvor.
    Api.getFoundationState()
      .then(setFoundation)
      .catch(() => setFoundation(null));
  }, []);

  const loadMonth = useCallback(async () => {
    try {
      setMonthState(await Api.getMonthCloseState(month));
      setMonthError('');
    } catch (e) {
      // Kein Toast und kein eigener Satz: das Backend sagt, warum es keinen
      // Stand gibt — ein Monat außerhalb des Geschäftsjahres nennt das Jahr, zu
      // dem er gehört, ein Lesefehler nennt sich selbst. Ein pauschales „gehört
      // zu einem anderen Geschäftsjahr" wäre bei jedem zweiten Grund falsch.
      setMonthState(null);
      setMonthError(e instanceof Error ? e.message : String(e));
    }
  }, [month]);

  useEffect(() => {
    void loadMonth();
  }, [loadMonth]);

  const total = tasks ? tasks.overdue.length + tasks.open.length + tasks.upcoming.length : 0;
  const monthSteps = monthState?.steps ?? [];
  const monthDone = monthSteps.filter((step) => step.state === 'done').length;
  const months = monthOptions(month);
  // „August 2026" und nicht „2026-08": der Schlüssel ist die Kennung des
  // Zeitraums und keine Bezeichnung. Genommen wird der Name aus der Auswahl,
  // damit beide dasselbe sagen.
  const monthLabel = months.find((option) => option.value === month)?.label ?? month;

  // Der Zustand zwischen Beurkundung und Eintragung. Dieselbe Bedingung wie am
  // Gründungsabschnitt der Fristenseite: er endet mit der Eintragung, und der
  // Streifen endet mit ihm.
  const inGruendung =
    foundation?.applies === true &&
    foundation.hasFoundation &&
    foundation.stage === 'vorgesellschaft' &&
    Boolean(foundation.foundation?.notarizedOn);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Aufgaben"
        context={
          tasks ? `Stand ${formatDate(tasks.today)} · ${total} offen` : 'Was heute zu tun ist'
        }
        action={
          /* Genau eine Primäraktion je Ansicht (§10.4). Den Monatsabschluss
             öffnet der Abschnitt weiter unten, der auch seinen Stand zeigt —
             ein zweiter Knopf dafür stünde hier ohne den Monat, den er meint. */
          <div className="flex gap-2">
            <Button
              variant="secondary"
              icon={<Landmark className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => onNavigate('bank')}
            >
              Bankumsätze abgleichen
            </Button>
            <Button
              variant="primary"
              icon={<FileText className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => onNavigate('invoices')}
            >
              Neue Rechnung
            </Button>
          </div>
        }
      />

      {inGruendung && (
        <div className="mt-8">
          <Notice
            text={
              <span className="flex items-center">
                <span>
                  {`Ihre Gesellschaft ist seit dem ${formatDate(
                    foundation!.foundation!.notarizedOn,
                  )} in Gründung${
                    foundation!.guide?.nextTitle
                      ? ` — als Nächstes steht an: ${foundation!.guide.nextTitle}`
                      : ' — bis zur Eintragung haften die Handelnden persönlich'
                  }.`}
                </span>
                <GruendungHelpMark onMore={() => setGruendungHelp(true)} />
              </span>
            }
            action={
              <Button variant="secondary" size="sm" onClick={() => onNavigate('gruendung')}>
                Zum Gründungsweg
              </Button>
            }
          />
        </div>
      )}

      {loading ? (
        <div className="mt-8">
          <SkeletonRows rows={6} />
        </div>
      ) : total === 0 ? (
        <div className="mt-8">
          <EmptyState
            title="Nichts offen"
            description="Keine verstrichene Frist, keine Belege ohne Buchung, keine Bankumsätze ohne Zuordnung."
          />
        </div>
      ) : (
        GROUPS.filter((group) => (tasks?.[group.key] ?? []).length > 0).map((group, index) => (
          <Section helpSummary="Die Liste zeigt offene Aufgaben aus Ihrer Buchhaltung und aktualisiert sich mit Ihrer Arbeit."
            key={group.key}
            title={group.title}
            context={group.context}
            divider={index > 0}
            className={index === 0 ? 'mt-8' : undefined}
            explain={
              index === 0 ? (
                <>
                  Die Liste entsteht aus den Daten: Befunde der Prüfläufe, Fristen, Bankumsätze ohne
                  Zuordnung, Belege ohne Buchung, überfällige Forderungen, ablaufende Bescheinigungen
                  und die Sicherung. Was erledigt ist, verschwindet von selbst — abgehakt wird hier
                  nichts. Warum eine Zeile dasteht, steht hinter dem Fragezeichen daneben.
                </>
              ) : undefined
            }
          >
            <Table>
              <Thead>
                <Tr>
                  <Th>Aufgabe</Th>
                  <Th className="w-72">Kontext</Th>
                  <Th className="w-32" aria-label="Ziel" />
                </Tr>
              </Thead>
              <Tbody>
                {(tasks?.[group.key] ?? []).map((task) => {
                  const tab = targetTab(task.target?.page ?? '');
                  return (
                    <Tr key={`${task.key}-${task.title}`}>
                      <Td className="max-w-[32rem]">
                        <span className="flex items-center gap-1.5">
                          <span
                            className={cn('mark-diamond shrink-0', group.mark)}
                            aria-hidden="true"
                          />
                          <span className="truncate">{task.title}</span>
                          {/* Der Grund gehört hinter das Erklärzeichen und nicht
                              in die Zeile: eine Tabellenzelle hat keinen
                              Erklärtext (§15.1). */}
                          {task.reference ? (
                            <Help summary="Hier erfahren Sie, warum diese Aufgabe ansteht und was zu erledigen ist." label={`Erklärung zu ${task.title}`}>
                              {task.why} {task.reference}
                            </Help>
                          ) : (
                            <Help summary="Hier erfahren Sie, warum diese Aufgabe ansteht und was zu erledigen ist." label={`Erklärung zu ${task.title}`}>
                              {task.why}
                            </Help>
                          )}
                        </span>
                      </Td>
                      <Td className="text-ink-subtle text-caption num">{taskContext(task)}</Td>
                      <Td className="text-right pl-0">
                        {tab && (
                          <Button
                            variant="quiet"
                            size="sm"
                            onClick={() => onNavigate(tab, targetParams(task))}
                          >
                            {targetLabel(tab)}
                          </Button>
                        )}
                      </Td>
                    </Tr>
                  );
                })}
              </Tbody>
            </Table>
          </Section>
        ))
      )}

      <Section
        title="Monatsabschluss"
        context={
          monthState
            ? `${monthState.label} · ${monthDone} von ${monthSteps.length} Schritten erledigt`
            : 'Prüfbericht, Festschreibung, Voranmeldung'
        }
        action={
          <Button variant="secondary" onClick={() => setMonthOpen(true)}>
            Monat öffnen
          </Button>
        }
      >
        {monthState === null ? (
          <p className="text-body text-ink-muted">
            {`Für ${monthLabel} liegt kein Stand vor.`}
            {monthError ? ` ${monthError}` : ''}
          </p>
        ) : (
          <ul className="flex flex-col">
            {monthSteps.map((step) => (
              <li
                key={step.key}
                className="flex items-start gap-3 py-2 border-t border-line first:border-t-0"
              >
                <span
                  className={cn(
                    'mark-diamond mt-1.5 shrink-0',
                    step.state === 'done'
                      ? 'bg-positive'
                      : step.state === 'blocked'
                        ? 'bg-negative'
                        : step.state === 'open'
                          ? 'bg-attention'
                          : 'bg-ink-faint',
                  )}
                  aria-hidden="true"
                />
                <span className="flex-1 min-w-0">
                  <span className="block text-body text-ink">{step.title}</span>
                  <span className="block text-caption text-ink-muted">{step.note}</span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>

      {summary && (
        <Section title="Übersicht" context="Stand des laufenden Geschäftsjahres">
          <StatRow>
            <Stat
              label={
                <>
                  Bankguthaben
                  <Help summary="Der Betrag zeigt den gebuchten Stand Ihres Geschäftskontos." label="Erklärung zum Bankguthaben">
                    Aktueller Gesamtsaldo auf dem Geschäftskonto.
                  </Help>
                </>
              }
              value={formatCents(summary.bankBalance)}
              context="Geschäftskonto 1800"
            />
            <Stat
              label="Einnahmen"
              value={formatCents(summary.totalRevenue)}
              context="Gesamterlöse"
            />
            <Stat
              label="Ausgaben"
              value={formatCents(summary.totalExpenses)}
              context="Betriebsausgaben"
            />
            <Stat
              label="Ergebnis"
              value={formatCents(summary.netIncome)}
              context="vor Steuern"
              tone={summary.netIncome >= 0 ? 'positive' : 'negative'}
            />
          </StatRow>
        </Section>
      )}

      {recentEntries.length > 0 && (
        <Section
          title="Zuletzt erfasst"
          context="Die acht zuletzt gebuchten Vorgänge"
          action={
            <Button variant="quiet" onClick={() => onNavigate('journal')}>
              Zum Journal
            </Button>
          }
        >
          <Table>
            <Thead>
              <Tr>
                <Th>Beleg</Th>
                <Th>Datum</Th>
                <Th>Buchungstext</Th>
                <Th>Konten</Th>
                <Th numeric>Betrag</Th>
                <Th>Status</Th>
              </Tr>
            </Thead>
            <Tbody>
              {recentEntries.map((entry) => {
                const lines = entry.lines ?? [];
                const gross = lines
                  .filter((line) => line.side === 'S')
                  .reduce((sum, line) => sum + line.amount, 0);
                const isReversal = entry.kind === 'reversal';

                return (
                  <Tr key={entry.id} variant={isReversal ? 'storno' : 'default'}>
                    <Td code>{entry.entryNumber}</Td>
                    <Td className="text-ink-subtle num">{formatDate(entry.bookingDate)}</Td>
                    <Td className="max-w-[24rem] truncate" title={entry.description}>
                      {entry.description}
                    </Td>
                    <Td code>{lines.map((line) => line.account).join(' · ')}</Td>
                    <Td numeric>{formatCents(gross, entry.currency)}</Td>
                    <Td>
                      <StatusBadge status={isReversal ? 'storniert' : 'gebucht'} />
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        </Section>
      )}

      <GruendungHelpDialog open={gruendungHelp} onClose={() => setGruendungHelp(false)} />

      <MonthCloseDialog
        open={monthOpen}
        month={month}
        initialState={monthState}
        onMonthChange={setMonth}
        months={months}
        onClose={() => setMonthOpen(false)}
        onChanged={async () => {
          await loadMonth();
          await loadTasks();
        }}
        onNavigate={onNavigate}
      />
    </div>
  );
};
