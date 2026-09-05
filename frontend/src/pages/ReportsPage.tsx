import React, { useEffect, useState } from 'react';
import { Download, Table2 } from 'lucide-react';
import { CompanySettings, FinancialStatement, VatSummary } from '../types';
import { Api } from '../services/api';
import { formatCents } from '../utils/formatters';
import type { NavigateFn } from '../components/Sidebar';
import {
  DepthChoice,
  StatementTab,
  StatementView,
} from '../components/StatementView';
import {
  Button,
  EmptyState,
  HelpPopover,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

/**
 * Auswertungen: Bilanz, Gewinn- und Verlustrechnung und Umsatzsteuer.
 *
 * Die Zahlen kommen fertig aus dem Backend. Bilanz und GuV entstanden bis
 * Welle 2 hier durch Filtern nach Kontenklasse und waren damit eine zweite,
 * abweichende Wahrheit; jetzt liest die Ansicht die Gliederung nach den
 * §§ 266 und 275 HGB aus `GetStatement` und rechnet nichts nach. Dasselbe gilt
 * für die Umsatzsteuer, an deren Buchungszeilen der Steuerschlüssel und die
 * Bemessungsgrundlage hängen.
 */

type Tab = StatementTab | 'ust';

/** Die Reiter, die den Abschluss zeigen — sie tragen Tiefe und Ausgabe. */
const STATEMENT_TABS: Tab[] = ['bilanz', 'guv', 'angaben', 'klasse'];

const MONTH_NAMES = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];

const QUARTER_LABELS = ['Jan–Mär', 'Apr–Jun', 'Jul–Sep', 'Okt–Dez'];

export interface ReportsPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile. Der Abschluss folgt ihm. */
  year: number;
  /** Weg von der Gliederungszeile über das Konto ins Kontoblatt (GOB-02). */
  onNavigate?: NavigateFn;
}

export const ReportsPage: React.FC<ReportsPageProps> = ({ year, onNavigate }) => {
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [vatByPeriod, setVatByPeriod] = useState<Record<string, VatSummary>>({});
  const [loading, setLoading] = useState(true);

  const [selectedQuarter, setSelectedQuarter] = useState<number>(1);
  const [selectedMonth, setSelectedMonth] = useState<number>(1);

  const [tab, setTab] = useState<Tab>('bilanz');
  const [statement, setStatement] = useState<FinancialStatement | null>(null);
  const [depth, setDepth] = useState<DepthChoice>('auto');
  const [loadingStatement, setLoadingStatement] = useState(true);
  const [statementError, setStatementError] = useState<string | null>(null);
  const [exporting, setExporting] = useState<'pdf' | 'csv' | null>(null);

  // Die Voranmeldung wird beim Jahreswechsel neu gelesen: die Reiter dieser
  // Seite zeigen dasselbe Geschäftsjahr, sonst stünde neben der Bilanz 2026 die
  // Umsatzsteuer des Vorjahres.
  useEffect(() => {
    void loadData();
  }, [year]);

  useEffect(() => {
    void loadStatement();
    // Die Tiefe ist ein Parameter des Aufbaus, kein Filter der Ansicht: eine
    // verkürzte Bilanz wird gebaut und nicht ausgeblendet.
  }, [year, depth]);

  async function loadStatement() {
    setLoadingStatement(true);
    try {
      setStatement(await Api.getStatement(year, depth === 'auto' ? '' : depth));
      setStatementError(null);
    } catch (e) {
      // Ein Befund aus dem Aufstellen — etwa eine Bilanz, die nicht aufgeht —
      // ist ein Fehler und kein Leerzustand: er gehört als Hinweisfläche über
      // die Ansicht (§10), damit der Satz des Backends lesbar bleibt.
      setStatement(null);
      setStatementError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoadingStatement(false);
    }
  }

  async function handleExport(kind: 'pdf' | 'csv') {
    setExporting(kind);
    try {
      if (kind === 'pdf') {
        const base64 = await Api.exportStatementPDF(year);
        downloadBlob(
          `jahresabschluss-${year}.pdf`,
          new Blob([bufferFromBase64(base64)], { type: 'application/pdf' }),
        );
      } else {
        const csv = await Api.exportStatementCSV(year);
        // Das Byte-Order-Mark steht davor, damit die Tabellenkalkulation die
        // Umlaute als UTF-8 liest und nicht als Zeichensalat.
        downloadBlob(
          `jahresabschluss-${year}.csv`,
          new Blob([`﻿${csv}`], { type: 'text/csv;charset=utf-8' }),
        );
      }
      toast.success(kind === 'pdf' ? 'Abschluss als PDF gespeichert.' : 'Gliederung als CSV gespeichert.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setExporting(null);
    }
  }

  async function loadData() {
    setLoading(true);
    try {
      const cfg = await Api.getCompanySettings();
      setSettings(cfg);

      const year = cfg.fiscalYear || new Date().getFullYear();
      const periods: Array<[string, string, string]> = [['year', '', '']];
      for (let q = 1; q <= 4; q++) {
        periods.push([`q${q}`, `${year}-${String(q * 3 - 2).padStart(2, '0')}-01`, endOfMonth(year, q * 3)]);
      }
      for (let m = 1; m <= 12; m++) {
        periods.push([`m${m}`, `${year}-${String(m).padStart(2, '0')}-01`, endOfMonth(year, m)]);
      }
      const results = await Promise.all(periods.map(([, from, to]) => Api.getVatSummary(from, to)));
      setVatByPeriod(Object.fromEntries(periods.map(([key], i) => [key, results[i]])));

      const currentMonth = new Date().getMonth() + 1;
      setSelectedQuarter(Math.floor((currentMonth - 1) / 3) + 1);
      setSelectedMonth(currentMonth);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  const currentYear = settings?.fiscalYear || new Date().getFullYear();
  const vatPeriod = settings?.vatPeriod || 'quarter';

  /** Die Zahlen der Voranmeldung auf die Feldnamen der Ansicht gebracht. */
  const vatView = (key: string) => {
    const v = vatByPeriod[key];
    const groups = v?.taxableRevenue ?? [];
    const find = (rate: number) => groups.find((g) => g.rate === rate);
    const rev19 = find(1900);
    const rev7 = find(700);
    const exempt =
      (v?.exemptRevenue ?? 0) +
      (v?.intraCommunitySupply ?? 0) +
      (v?.export ?? 0) +
      (v?.reverseChargeSupply ?? 0);

    return {
      rev19Net: rev19?.net ?? 0,
      tax19: rev19?.tax ?? 0,
      rev7Net: rev7?.net ?? 0,
      tax7: rev7?.tax ?? 0,
      revExemptNet: exempt,
      totalRevenueNet: (rev19?.net ?? 0) + (rev7?.net ?? 0) + exempt,
      totalTax: v?.totalOwedTax ?? 0,
      inputTax: v?.inputTax ?? 0,
      zahllast: v?.payable ?? 0,
    };
  };

  const activeVat =
    vatPeriod === 'month'
      ? vatView(`m${selectedMonth}`)
      : vatPeriod === 'quarter'
        ? vatView(`q${selectedQuarter}`)
        : vatView('year');

  const quarters = [1, 2, 3, 4].map((q) => ({
    quarter: q,
    label: `Q${q} · ${QUARTER_LABELS[q - 1]}`,
    ...vatView(`q${q}`),
  }));

  const refund = activeVat.zahllast < 0;

  const onStatement = STATEMENT_TABS.includes(tab);

  /** Bilanz, Staffel, Angaben und Größenklasse — vier Sichten auf einen Aufbau. */
  const statementPanel = (view: StatementTab) => {
    if (loadingStatement) return <SkeletonRows rows={10} />;
    if (!statement) {
      // Der Fehler steht als Hinweisfläche über den Reitern; der Leerzustand
      // gilt allein dem Fall, dass es nichts zu zeigen gibt.
      return statementError ? null : (
        <EmptyState
          title={`Kein Abschluss für ${year}`}
          description="Das Backend hat keine Gliederung geliefert."
        />
      );
    }
    return (
      <StatementView
        data={statement}
        view={view}
        depth={depth}
        onDepthChange={setDepth}
        onNavigate={onNavigate}
      />
    );
  };

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Auswertungen"
        context={`Geschäftsjahr ${year} · berechnet aus den erfassten Buchungen`}
        action={
          onStatement ? (
            <div className="flex gap-2">
              <Button
                variant="secondary"
                icon={<Table2 className="w-4 h-4" strokeWidth={1.5} />}
                onClick={() => handleExport('csv')}
                loading={exporting === 'csv'}
                disabled={!statement || exporting !== null}
              >
                Als CSV
              </Button>
              <Button
                variant="primary"
                icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
                onClick={() => handleExport('pdf')}
                loading={exporting === 'pdf'}
                disabled={!statement || exporting !== null}
              >
                Als PDF
              </Button>
            </div>
          ) : undefined
        }
      />

      {onStatement && statementError && (
        <Notice
          className="mt-6"
          tone="negative"
          text={statementError}
          action={
            <Button variant="secondary" size="sm" onClick={() => void loadStatement()}>
              Erneut aufstellen
            </Button>
          }
        />
      )}

      <Tabs<Tab>
        items={[
          { value: 'bilanz', label: 'Bilanz' },
          { value: 'guv', label: 'Gewinn- und Verlustrechnung' },
          { value: 'angaben', label: 'Angaben unter der Bilanz' },
          { value: 'klasse', label: 'Größenklasse und Fristen' },
          { value: 'ust', label: 'Umsatzsteuer' },
        ]}
        value={tab}
        onValueChange={setTab}
        className="mt-6"
      >
        <TabPanel value="bilanz">{statementPanel('bilanz')}</TabPanel>
        <TabPanel value="guv">{statementPanel('guv')}</TabPanel>
        <TabPanel value="angaben">{statementPanel('angaben')}</TabPanel>
        <TabPanel value="klasse">{statementPanel('klasse')}</TabPanel>

        {/* Die Umsatzsteuer-Übersicht steht unverändert, wie vor Welle 2. */}
        <TabPanel value="ust">
          {loading ? (
            <SkeletonRows rows={8} />
          ) : (
            <>
              <Section
                title={vatPeriod === 'month' ? 'Monatliche Voranmeldung' : 'Voranmeldung je Quartal'}
                context={`Geschäftsjahr ${currentYear}`}
                divider={false}
                action={
                  vatPeriod === 'month' ? (
                    <Select
                      items={MONTH_NAMES.map((name, index) => ({
                        value: index + 1,
                        label: `${name} ${currentYear}`,
                      }))}
                      value={selectedMonth}
                      onValueChange={setSelectedMonth}
                      className="w-48"
                    />
                  ) : (
                    <Select
                      items={quarters.map((q) => ({ value: q.quarter, label: q.label }))}
                      value={selectedQuarter}
                      onValueChange={setSelectedQuarter}
                      className="w-48"
                    />
                  )
                }
              >
                <StatRow>
                  <Stat
                    label="Umsätze netto"
                    value={formatCents(activeVat.totalRevenueNet)}
                    context="im gewählten Zeitraum"
                  />
                  <Stat
                    label="Umsatzsteuer"
                    value={formatCents(activeVat.totalTax)}
                    context="19 % und 7 % auf Erlöse"
                  />
                  <Stat
                    label="Abziehbare Vorsteuer"
                    value={formatCents(activeVat.inputTax)}
                    context="aus Betriebsausgaben"
                  />
                  <Stat
                    label={refund ? 'Erstattungsanspruch' : 'Zahllast'}
                    value={formatCents(Math.abs(activeVat.zahllast))}
                    context={refund ? 'Guthaben beim Finanzamt' : 'an das Finanzamt zu zahlen'}
                    tone={refund ? 'positive' : 'neutral'}
                  />
                </StatRow>
              </Section>

              <Section
                title="Kennziffern der Voranmeldung"
                context={
                  vatPeriod === 'month'
                    ? `${MONTH_NAMES[selectedMonth - 1]} ${currentYear}`
                    : `Q${selectedQuarter} ${currentYear}`
                }
                action={
                  <HelpPopover label="Erklärung zu den Kennziffern">
                    Die Kennziffern entsprechen den Feldern des amtlichen Vordrucks der
                    Umsatzsteuer-Voranmeldung. Buchfink übermittelt nicht selbst: Die Zahlen werden in
                    Mein ELSTER übertragen oder an die Steuerberatung übergeben.
                  </HelpPopover>
                }
              >
                <Table>
                  <Thead>
                    <Tr>
                      <Th className="w-20">Kz</Th>
                      <Th>Position</Th>
                      <Th numeric className="w-40">
                        Bemessung
                      </Th>
                      <Th numeric className="w-40">
                        Steuer
                      </Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    <Tr>
                      <Td code>81</Td>
                      <Td>Steuerpflichtige Umsätze zum Steuersatz von 19 %</Td>
                      <Td numeric className="text-ink-muted">
                        {formatCents(activeVat.rev19Net)}
                      </Td>
                      <Td numeric>{formatCents(activeVat.tax19)}</Td>
                    </Tr>
                    <Tr>
                      <Td code>86</Td>
                      <Td>Steuerpflichtige Umsätze zum Steuersatz von 7 %</Td>
                      <Td numeric className="text-ink-muted">
                        {formatCents(activeVat.rev7Net)}
                      </Td>
                      <Td numeric>{formatCents(activeVat.tax7)}</Td>
                    </Tr>
                    <Tr>
                      <Td code>66</Td>
                      <Td>Abziehbare Vorsteuerbeträge aus Rechnungen anderer Unternehmen</Td>
                      <Td numeric className="text-ink-muted">
                        —
                      </Td>
                      <Td numeric>− {formatCents(activeVat.inputTax)}</Td>
                    </Tr>
                    <Tr variant="sum">
                      <Td code>83</Td>
                      <Td>Verbleibende Umsatzsteuer-Vorauszahlung</Td>
                      <Td />
                      <Td numeric className={refund ? 'text-positive-text' : undefined}>
                        {formatCents(activeVat.zahllast)}
                      </Td>
                    </Tr>
                  </Tbody>
                </Table>
              </Section>

              {vatPeriod === 'quarter' && (
                <Section title="Jahresverlauf" context={`Alle vier Quartale des Jahres ${currentYear}`}>
                  <Table>
                    <Thead>
                      <Tr>
                        <Th>Zeitraum</Th>
                        <Th numeric className="w-40">
                          Umsatz netto
                        </Th>
                        <Th numeric className="w-40">
                          Umsatzsteuer
                        </Th>
                        <Th numeric className="w-40">
                          Vorsteuer
                        </Th>
                        <Th numeric className="w-44">
                          Zahllast
                        </Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {quarters.map((q) => (
                        <Tr key={q.quarter} variant={q.quarter === selectedQuarter ? 'selected' : 'default'}>
                          <Td>{q.label}</Td>
                          <Td numeric>{formatCents(q.totalRevenueNet)}</Td>
                          <Td numeric>{formatCents(q.totalTax)}</Td>
                          <Td numeric>{formatCents(q.inputTax)}</Td>
                          <Td numeric className={q.zahllast < 0 ? 'text-positive-text' : undefined}>
                            {formatCents(q.zahllast)}
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </Section>
              )}
            </>
          )}
        </TabPanel>
      </Tabs>
    </div>
  );
};

// -------------------------------------------------------------------------

/** Base64 aus der Bridge in Bytes — das PDF kommt wie der Rechnungsexport. */
function bufferFromBase64(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const buffer = new ArrayBuffer(binary.length);
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return buffer;
}

function downloadBlob(name: string, blob: Blob) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/** Letzter Tag eines Monats als ISO-Datum. */
function endOfMonth(year: number, month: number): string {
  const last = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return `${year}-${String(month).padStart(2, '0')}-${String(last).padStart(2, '0')}`;
}
