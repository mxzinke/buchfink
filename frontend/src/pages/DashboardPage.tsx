import React, { useEffect, useState } from 'react';
import { FileText, Landmark } from 'lucide-react';
import { CompanySettings, FinancialSummary, JournalEntry } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import type { NavigateFn } from '../components/Sidebar';
import {
  Button,
  EmptyState,
  PageHeader,
  Section,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from '../components/ui';

interface DashboardPageProps {
  onNavigate: NavigateFn;
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
      {/* Die Kennzahlen — Bankguthaben, Einnahmen, Ausgaben, Ergebnis — stehen
          auf der Aufgabenseite unter der Liste (Architektur 6.1). Zweimal
          dieselben vier Zahlen an zwei Stellen sind keine zweite Auskunft,
          sondern zwei Orte, an denen dieselbe Frage beantwortet wird — und
          einer davon ist irgendwann veraltet. Hier bleibt, was die
          Aufgabenliste nicht zeigt: die zuletzt erfassten Vorgänge. */}
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
