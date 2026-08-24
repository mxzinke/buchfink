import React, { useEffect, useState } from 'react';
import { FileText, Landmark } from 'lucide-react';
import { CompanySettings, FinancialSummary, JournalEntry } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import {
  Button,
  EmptyState,
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
} from '../components/ui';

interface DashboardPageProps {
  onNavigate: (tab: any) => void;
}

export const DashboardPage: React.FC<DashboardPageProps> = ({ onNavigate }) => {
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [recentEntries, setRecentEntries] = useState<JournalEntry[]>([]);
  const [settings, setSettings] = useState<CompanySettings | null>(null);
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
    } finally {
      setLoading(false);
    }
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
                    const gross = entry.lines
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
                        <Td code>{entry.lines.map((line) => line.account).join(' · ')}</Td>
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
