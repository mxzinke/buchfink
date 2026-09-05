import React, { useEffect, useState } from 'react';
import { FileText, Landmark } from 'lucide-react';
import { CheckRun, CompanySettings, Deadline, FinancialSummary, JournalEntry } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import type { NavigateFn, TabType } from '../components/Sidebar';
import {
  Button,
  EmptyState,
  HelpPopover,
  HelpTooltip,
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
} from '../components/ui';

interface DashboardPageProps {
  onNavigate: NavigateFn;
}

/**
 * Eine Zeile der Aufgabenliste: was zu tun ist, woher es kommt und wohin es
 * führt. Eine Aufgabe ohne Weg dorthin wäre eine Hausaufgabe ohne Adresse.
 */
interface Task {
  key: string;
  title: string;
  context: string;
  /** `attention` heißt: es ist zu erledigen. `negative` heißt: die Frist ist um. */
  tone: 'attention' | 'negative';
  target: TabType;
  action: string;
}

export const DashboardPage: React.FC<DashboardPageProps> = ({ onNavigate }) => {
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [recentEntries, setRecentEntries] = useState<JournalEntry[]>([]);
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [sum, bookings, cfg] = await Promise.all([
        Api.getFinancialSummary(),
        Api.getJournalEntries(),
        Api.getCompanySettings(),
      ]);
      setSummary(sum);
      setRecentEntries(bookings.slice(-8).reverse());
      setSettings(cfg);
      await loadTasks(cfg.fiscalYear || new Date().getFullYear());
    } finally {
      setLoading(false);
    }
  };

  /**
   * Die Aufgabenliste. Sie rechnet nichts nach: die Fristen kommen aus
   * `GetDeadlines`, die Befunde aus dem letzten Prüflauf, die Zahlen aus den
   * Belegen und Bankumsätzen. Was erledigt ist, steht dort schon als erledigt.
   */
  const loadTasks = async (year: number) => {
    const todayIso = new Date().toISOString().slice(0, 10);
    const next: Task[] = [];
    try {
      // Die Termine des Vorjahres gehören dazu, soweit sie in diesem Jahr fällig
      // werden: die Voranmeldung für Dezember ist am 10. Januar fällig, die des
      // vierten Quartals am 10. Februar, die Festschreibung des Dezembers Ende
      // Januar. Sie sind Termine des Vorjahres und stünden sonst im Januar und
      // Februar nirgends — genau dann, wenn sie am häufigsten überfällig sind.
      const [current, prior] = await Promise.all([
        Api.getDeadlines(year),
        Api.getDeadlines(year - 1),
      ]);
      const carried = prior.filter((d) => d.dueDate && d.dueDate >= `${year}-01-01`);

      // Ein Schlüssel kommt nur einmal vor: die Gründungspflichten tragen kein
      // Jahr im Schlüssel und stünden sonst doppelt auf der Liste.
      const seen = new Set<string>();
      const overdue: Deadline[] = [];
      for (const d of [...current, ...carried]) {
        if (seen.has(d.key)) continue;
        seen.add(d.key);
        if (d.isDone || !d.dueDate || d.dueDate >= todayIso) continue;
        overdue.push(d);
      }
      overdue.sort((a, b) => a.dueDate.localeCompare(b.dueDate));

      for (const d of overdue) {
        next.push({
          key: `deadline-${d.key}`,
          title: d.title,
          context: `Fällig am ${formatDate(d.dueDate)} · ${d.reference}`,
          tone: 'negative',
          target: d.key.startsWith('ustva.') || d.key.startsWith('zm.') ? 'vat' : 'deadlines',
          action: 'Hin dazu',
        });
      }
    } catch (e) {
      console.error(e);
    }

    try {
      // Nachträge sind Buchungen, deren Voranmeldungszeitraum bereits übermittelt
      // ist. Sie verlangen eine berichtigte Anmeldung (§ 153 AO) und stehen sonst
      // nur auf dem Blatt der Umsatzsteuer — wo niemand nachsieht, der nicht
      // ohnehin dort ist.
      const vatPeriods = await Api.getVatPeriods(year);
      const running =
        vatPeriods.find((p) => p.from <= todayIso && p.to >= todayIso) ??
        vatPeriods[vatPeriods.length - 1];
      if (running) {
        const draft = await Api.getVatReturn(running.key);
        // Ohne Nachtrag liefert das Backend keine Liste, sondern `null`. Der
        // Fehler fiele hier in das try/catch und stünde nur in der Konsole —
        // die Aufgabe fehlte still. Deshalb der Wert davor.
        const lateEntries = draft.lateEntries ?? [];
        if (lateEntries.length > 0) {
          next.push({
            key: 'vat-late-entries',
            title: `${lateEntries.length} Nachträge zu übermittelten Zeiträumen`,
            context: 'Buchungen, für die eine berichtigte Voranmeldung fällig ist',
            tone: 'attention',
            target: 'vat',
            action: 'Zur Umsatzsteuer',
          });
        }
      }
    } catch (e) {
      console.error(e);
    }

    try {
      const receipts = await Api.getReceipts('filed');
      if (receipts.length > 0) {
        next.push({
          key: 'receipts-open',
          title: `${receipts.length} Belege ohne Buchung`,
          context: 'abgelegt, aber noch nicht gebucht',
          tone: 'attention',
          target: 'receipts',
          action: 'Zu den Belegen',
        });
      }
    } catch (e) {
      console.error(e);
    }

    try {
      const unmatched = (await Api.getBankTransactions()).filter(
        (tx) => tx.matchStatus === 'unmatched',
      );
      if (unmatched.length > 0) {
        next.push({
          key: 'bank-unmatched',
          title: `${unmatched.length} Bankumsätze ohne Zuordnung`,
          context: 'ohne Zuordnung bleibt die Kasse nicht abgestimmt',
          tone: 'attention',
          target: 'bank',
          action: 'Zum Abgleich',
        });
      }
    } catch (e) {
      console.error(e);
    }

    try {
      const runs: CheckRun[] = await Api.getCheckRuns(0);
      const latest = runs[0];
      const latestFindings = latest?.findings ?? [];
      if (latest && latestFindings.length > 0) {
        const blocking = latestFindings.filter((f) => f.severity === 'blocking').length;
        next.push({
          key: 'check-findings',
          title: `${latestFindings.length} Befunde aus dem letzten Prüflauf`,
          context: `${blocking} blockierend · Stichtag ${formatDate(latest.cutoffDate)}`,
          tone: blocking > 0 ? 'negative' : 'attention',
          target: 'audit',
          action: 'Zum Prüfbericht',
        });
      }
    } catch (e) {
      console.error(e);
    }

    setTasks(next);
  };

  const context = [
    settings?.taxationType ? `${settings.taxationType}-Versteuerung` : null,
    'lokal gespeichert',
  ]
    .filter(Boolean)
    .join(' · ');

  const hasData =
    recentEntries.length > 0 || (summary?.totalRevenue ?? 0) > 0 || (summary?.totalExpenses ?? 0) > 0;

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Buchhaltungsübersicht"
        context={context}
        action={
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

      {loading || !summary ? (
        <div className="mt-8">
          <SkeletonRows rows={6} />
        </div>
      ) : (
        <>
          <Section
            title="Aufgaben"
            context={tasks.length > 0 ? `${tasks.length} offen` : 'nichts offen'}
            divider={false}
            className="mt-8"
            action={
              <HelpPopover label="Erklärung zur Aufgabenliste">
                Die Liste entsteht aus den Daten: überfällige Termine aus der Fristenübersicht —
                auch die des Vorjahres, die erst in diesem Jahr fällig wurden —, Nachträge zu
                übermittelten Voranmeldungszeiträumen, abgelegte Belege ohne Buchung, Bankumsätze
                ohne Zuordnung und die Befunde des letzten Prüflaufs. Was erledigt ist,
                verschwindet von selbst — abgehakt wird hier nichts.
              </HelpPopover>
            }
          >
            {tasks.length === 0 ? (
              <EmptyState
                title="Nichts offen"
                description="Keine überfälligen Termine, keine Belege ohne Buchung, keine Bankumsätze ohne Zuordnung."
              />
            ) : (
              <Table>
                <Thead>
                  <Tr>
                    <Th>Aufgabe</Th>
                    <Th className="w-96">Herkunft</Th>
                    <Th className="w-40" aria-label="Aktion" />
                  </Tr>
                </Thead>
                <Tbody>
                  {tasks.map((task) => (
                    <Tr key={task.key}>
                      <Td className="max-w-[28rem]">
                        <span className="flex items-center gap-2">
                          <span
                            className={cn(
                              'mark-diamond',
                              task.tone === 'negative' ? 'bg-negative' : 'bg-attention',
                            )}
                            aria-hidden="true"
                          />
                          <span className="truncate">{task.title}</span>
                        </span>
                      </Td>
                      <Td className="text-ink-subtle text-caption whitespace-normal">
                        {task.context}
                      </Td>
                      <Td className="text-right">
                        <Button variant="quiet" size="sm" onClick={() => onNavigate(task.target)}>
                          {task.action}
                        </Button>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </Section>

          <div className="mt-6">
            <StatRow>
              <Stat
                label={
                  <>
                    Bankguthaben
                    <HelpTooltip
                      label="Erklärung zum Bankguthaben"
                      content="Aktueller Gesamtsaldo auf dem Geschäftskonto."
                    />
                  </>
                }
                value={formatCents(summary.bankBalance)}
                context="Geschäftskonto 1800"
              />
              <Stat
                label={
                  <>
                    Einnahmen
                    <HelpTooltip
                      label="Erklärung zu den Einnahmen"
                      content="Summe aller Erlöse im laufenden Geschäftsjahr, vor Steuern."
                    />
                  </>
                }
                value={formatCents(summary.totalRevenue)}
                context="Gesamterlöse"
              />
              <Stat
                label={
                  <>
                    Ausgaben
                    <HelpTooltip
                      label="Erklärung zu den Ausgaben"
                      content="Summe aller Betriebsausgaben im laufenden Geschäftsjahr."
                    />
                  </>
                }
                value={formatCents(summary.totalExpenses)}
                context="Betriebsausgaben"
              />
              <Stat
                label={
                  <>
                    Ergebnis
                    <HelpTooltip
                      label="Erklärung zum Ergebnis"
                      content="Vorläufiger Gewinn oder Verlust vor Steuern, also Einnahmen minus Ausgaben."
                    />
                  </>
                }
                value={formatCents(summary.netIncome)}
                context="vor Steuern"
                tone={summary.netIncome >= 0 ? 'positive' : 'negative'}
              />
            </StatRow>
          </div>

          {!hasData ? (
            <div className="mt-8">
              <EmptyState
                title="Noch keine Buchungen erfasst"
                description="Buchungen entstehen aus dem Abgleich von Bankumsätzen mit Belegen oder direkt im Journal."
                action={
                  <>
                    <Button
                      variant="primary"
                      icon={<Landmark className="w-4 h-4" strokeWidth={1.5} />}
                      onClick={() => onNavigate('bank')}
                    >
                      Kontoauszug importieren
                    </Button>
                    <Button variant="secondary" onClick={() => onNavigate('journal')}>
                      Zum Journal
                    </Button>
                  </>
                }
              />
            </div>
          ) : (
            <Section
              title="Letzte Buchungen"
              context="Die acht zuletzt erfassten Vorgänge"
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
        </>
      )}
    </div>
  );
};
