import React, { useEffect, useState } from 'react';
import { Account, FinancialSummary, CompanySettings, VatSummary } from '../types';
import { Api } from '../services/api';
import { formatCents } from '../utils/formatters';
import {
  EmptyState,
  HelpPopover,
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
 * Auswertungen: GuV, Bilanz und Umsatzsteuer.
 *
 * Die Zahlen kommen fertig aus dem Backend, wo an jeder Buchungszeile der
 * Steuerschlüssel und die Bemessungsgrundlage hängen. Sie hier aus
 * Kontonummern zu rekonstruieren wäre eine zweite, abweichende Wahrheit.
 */

type Tab = 'guv' | 'bilanz' | 'ust';

const MONTH_NAMES = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];

const QUARTER_LABELS = ['Jan–Mär', 'Apr–Jun', 'Jul–Sep', 'Okt–Dez'];

export const ReportsPage: React.FC = () => {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [vatByPeriod, setVatByPeriod] = useState<Record<string, VatSummary>>({});
  const [tab, setTab] = useState<Tab>('guv');
  const [loading, setLoading] = useState(true);

  const [selectedQuarter, setSelectedQuarter] = useState<number>(1);
  const [selectedMonth, setSelectedMonth] = useState<number>(1);

  useEffect(() => {
    void loadData();
  }, []);

  async function loadData() {
    setLoading(true);
    try {
      const [accs, sum, cfg] = await Promise.all([
        Api.getAccounts(),
        Api.getFinancialSummary(),
        Api.getCompanySettings(),
      ]);
      setAccounts(accs);
      setSummary(sum);
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

  // Aufteilung nach SKR04 und HGB.
  const revenueAccounts = accounts.filter(
    (a) => (a.type === 'revenue' || a.kontenklasse === 4) && a.balance !== 0,
  );
  const expenseAccounts = accounts.filter(
    (a) =>
      (a.type === 'expense' || a.kontenklasse === 5 || a.kontenklasse === 6 || a.kontenklasse === 7) &&
      a.balance !== 0,
  );
  const assetAccounts = accounts.filter(
    (a) =>
      (a.type === 'asset' || a.balanceSide === 'Aktiva' || a.kontenklasse === 0 || a.kontenklasse === 1) &&
      a.balance !== 0,
  );
  const liabilityAccounts = accounts.filter(
    (a) =>
      (a.type === 'liability' ||
        a.type === 'equity' ||
        a.balanceSide === 'Passiva' ||
        a.kontenklasse === 2 ||
        a.kontenklasse === 3) &&
      a.balance !== 0,
  );

  const totalAssets = assetAccounts.reduce((sum, a) => sum + a.balance, 0);
  const totalLiabilities = liabilityAccounts.reduce((sum, a) => sum + a.balance, 0);

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

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Auswertungen"
        context={`Geschäftsjahr ${currentYear} · berechnet aus den erfassten Buchungen`}
      />

      <Tabs
        items={[
          { value: 'guv' as Tab, label: 'Gewinn & Verlust' },
          { value: 'bilanz' as Tab, label: 'Bilanz' },
          { value: 'ust' as Tab, label: 'Umsatzsteuer' },
        ]}
        value={tab}
        onValueChange={setTab}
        className="mt-6"
      >
        {/* ------------------------------------------------------------- */}
        <TabPanel value="guv">
          {loading ? (
            <SkeletonRows rows={8} />
          ) : (
            <Section
              title="Gewinn- und Verlustrechnung"
              context="Erlöse abzüglich Aufwendungen, vor Steuern"
              divider={false}
              action={
                <HelpPopover label="Erklärung zur Gewinn- und Verlustrechnung">
                  Die Aufstellung folgt dem Gesamtkostenverfahren nach § 275 HGB. Sie zeigt den
                  Stand des laufenden Geschäftsjahres und ist kein Jahresabschluss: Abgrenzungen,
                  Abschreibungen und Rückstellungen entstehen erst beim Abschluss.
                </HelpPopover>
              }
            >
              <Table>
                <Thead>
                  <Tr>
                    <Th className="w-28">Konto</Th>
                    <Th>Position</Th>
                    <Th numeric className="w-44">
                      Betrag
                    </Th>
                  </Tr>
                </Thead>
                <Tbody>
                  <GroupRow label="Erlöse" />
                  {revenueAccounts.length === 0 ? (
                    <Tr>
                      <Td className="text-ink-subtle" colSpan={3}>
                        Keine Erlöse gebucht
                      </Td>
                    </Tr>
                  ) : (
                    revenueAccounts.map((account) => (
                      <Tr key={account.number}>
                        <Td code>{account.number}</Td>
                        <Td>{account.name}</Td>
                        <Td numeric>{formatCents(account.balance)}</Td>
                      </Tr>
                    ))
                  )}
                  <Tr variant="sum">
                    <Td />
                    <Td>Summe Erlöse</Td>
                    <Td numeric>{formatCents(summary?.totalRevenue ?? 0)}</Td>
                  </Tr>

                  <GroupRow label="Aufwendungen" />
                  {expenseAccounts.length === 0 ? (
                    <Tr>
                      <Td className="text-ink-subtle" colSpan={3}>
                        Keine Aufwendungen gebucht
                      </Td>
                    </Tr>
                  ) : (
                    expenseAccounts.map((account) => (
                      <Tr key={account.number}>
                        <Td code>{account.number}</Td>
                        <Td>{account.name}</Td>
                        <Td numeric>{formatCents(account.balance)}</Td>
                      </Tr>
                    ))
                  )}
                  <Tr variant="sum">
                    <Td />
                    <Td>Summe Aufwendungen</Td>
                    <Td numeric>{formatCents(summary?.totalExpenses ?? 0)}</Td>
                  </Tr>

                  <GroupRow label="Ergebnis" />
                  <Tr variant="sum">
                    <Td />
                    <Td>Vorläufiges Jahresergebnis</Td>
                    <Td
                      numeric
                      className={
                        (summary?.netIncome ?? 0) >= 0 ? 'text-positive-text' : 'text-negative-text'
                      }
                    >
                      {formatCents(summary?.netIncome ?? 0)}
                    </Td>
                  </Tr>
                </Tbody>
              </Table>
            </Section>
          )}
        </TabPanel>

        {/* ------------------------------------------------------------- */}
        <TabPanel value="bilanz">
          {loading ? (
            <SkeletonRows rows={8} />
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              <BalanceSide
                title="Aktiva"
                context="Vermögen und Bankguthaben"
                accounts={assetAccounts}
                total={totalAssets}
              />
              <BalanceSide
                title="Passiva"
                context="Eigenkapital und Verbindlichkeiten"
                accounts={liabilityAccounts}
                total={totalLiabilities}
              />
            </div>
          )}
        </TabPanel>

        {/* ------------------------------------------------------------- */}
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

/** Zwischenüberschrift in der Aufstellung. Trägt allein durch Schriftschnitt. */
const GroupRow: React.FC<{ label: string }> = ({ label }) => (
  <Tr>
    <Td colSpan={3} className="text-overline uppercase text-ink-subtle">
      {label}
    </Td>
  </Tr>
);

const BalanceSide: React.FC<{
  title: string;
  context: string;
  accounts: Account[];
  total: number;
}> = ({ title, context, accounts, total }) => (
  <Section title={title} context={context} divider={false}>
    {accounts.length === 0 ? (
      <EmptyState title="Keine Positionen" />
    ) : (
      <Table>
        <Thead>
          <Tr>
            <Th className="w-24">Konto</Th>
            <Th>Position</Th>
            <Th numeric className="w-36">
              Betrag
            </Th>
          </Tr>
        </Thead>
        <Tbody>
          {accounts.map((account) => (
            <Tr key={account.number}>
              <Td code>{account.number}</Td>
              <Td className="max-w-[18rem] truncate" title={account.name}>
                {account.name}
              </Td>
              <Td numeric>{formatCents(account.balance)}</Td>
            </Tr>
          ))}
          <Tr variant="sum">
            <Td />
            <Td>Summe {title}</Td>
            <Td numeric>{formatCents(total)}</Td>
          </Tr>
        </Tbody>
      </Table>
    )}
  </Section>
);

/** Letzter Tag eines Monats als ISO-Datum. */
function endOfMonth(year: number, month: number): string {
  const last = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return `${year}-${String(month).padStart(2, '0')}-${String(last).padStart(2, '0')}`;
}
