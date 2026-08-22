import React, { useEffect, useState } from 'react';
import {
  TrendingUp,
  TrendingDown,
  ReceiptText,
} from 'lucide-react';
import { Account, FinancialSummary, CompanySettings, VatSummary } from '../types';
import { Api } from '../services/api';
import { formatCents } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const ReportsPage: React.FC = () => {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [vatByPeriod, setVatByPeriod] = useState<Record<string, VatSummary>>({});
  const [activeTab, setActiveTab] = useState<'guv' | 'bilanz' | 'ust'>('guv');

  // USt Period selection (adapted to client config)
  const [selectedQuarter, setSelectedQuarter] = useState<number>(1);
  const [selectedMonth, setSelectedMonth] = useState<number>(1);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [accs, sum, cfg] = await Promise.all([
        Api.getAccounts(),
        Api.getFinancialSummary(),
        Api.getCompanySettings(),
      ]);
      setAccounts(accs);
      setSummary(sum);
      setSettings(cfg);

      // Die Umsatzsteuerzahlen kommen aus dem Backend, wo an jeder Buchungszeile
      // der Steuerschlüssel und die Bemessungsgrundlage hängen. Sie hier aus
      // Kontonummern zu rekonstruieren wäre eine zweite, abweichende Wahrheit.
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

      // Default selected period to current quarter / month
      const curMonth = new Date().getMonth() + 1;
      const curQuarter = Math.floor((curMonth - 1) / 3) + 1;
      setSelectedQuarter(curQuarter);
      setSelectedMonth(curMonth);
    } catch (e) {
      console.error(e);
    }
  };

  const currentYear = settings?.fiscalYear || new Date().getFullYear();
  const vatPeriod = settings?.vatPeriod || 'quarter';

  // GuV & Bilanz account breakdowns according to SKR04 / HGB
  const revenueAccounts = accounts.filter(
    (a) => (a.type === 'revenue' || a.kontenklasse === 4) && a.balance !== 0
  );
  const expenseAccounts = accounts.filter(
    (a) =>
      (a.type === 'expense' || a.kontenklasse === 5 || a.kontenklasse === 6 || a.kontenklasse === 7) &&
      a.balance !== 0
  );
  const assetAccounts = accounts.filter(
    (a) => (a.type === 'asset' || a.balanceSide === 'Aktiva' || a.kontenklasse === 0 || a.kontenklasse === 1) && a.balance !== 0
  );
  const liabilityAccounts = accounts.filter(
    (a) =>
      (a.type === 'liability' || a.type === 'equity' || a.balanceSide === 'Passiva' || a.kontenklasse === 2 || a.kontenklasse === 3) &&
      a.balance !== 0
  );

  const totalAssets = assetAccounts.reduce((sum, a) => sum + a.balance, 0);
  const totalLiabilities = liabilityAccounts.reduce((sum, a) => sum + a.balance, 0);

  // Die Zahlen der Voranmeldung kommen fertig aus dem Backend; hier werden sie
  // nur auf die Feldnamen der Ansicht gebracht.
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
      reverseChargeTax: v?.reverseChargeTax ?? 0,
      intraCommunityTax: v?.intraCommunityAcquisitionTax ?? 0,
    };
  };

  const fullYearVat = vatView('year');
  const activePeriodVat =
    vatPeriod === 'month'
      ? vatView(`m${selectedMonth}`)
      : vatPeriod === 'quarter'
      ? vatView(`q${selectedQuarter}`)
      : fullYearVat;

  const quartersData = [1, 2, 3, 4].map((q) => ({
    quarter: q,
    label: `Q${q} (${q === 1 ? 'Jan–Mär' : q === 2 ? 'Apr–Jun' : q === 3 ? 'Jul–Sep' : 'Okt–Dez'})`,
    ...vatView(`q${q}`),
  }));

  const monthNames = [
    'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
    'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
  ];

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl sm:text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Auswertungen (GuV, Bilanz & USt)
            <HelpTooltip
              title="Auswertungen"
              content="Echtzeit-Berechnungen aus Ihren erfassten Buchungen für Gewinn & Verlust, Bilanzpositionen und die Umsatzsteuer."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Echtzeit-Berechnung direkt aus Ihren erfassten Buchungen &bull; Geschäftsjahr {currentYear}
          </p>
        </div>

        {/* View Switcher Tabs */}
        <div className="flex bg-stone-100 p-1 rounded-xl border border-stone-200 text-xs overflow-x-auto">
          <button
            onClick={() => setActiveTab('guv')}
            className={`px-4 py-1.5 rounded-lg font-medium transition-all whitespace-nowrap ${
              activeTab === 'guv'
                ? 'bg-amber-700 text-white shadow-xs font-semibold'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Gewinn- & Verlustrechnung (GuV)
          </button>
          <button
            onClick={() => setActiveTab('bilanz')}
            className={`px-4 py-1.5 rounded-lg font-medium transition-all whitespace-nowrap ${
              activeTab === 'bilanz'
                ? 'bg-amber-700 text-white shadow-xs font-semibold'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Bilanz (Vermögen & Schulden)
          </button>
          <button
            onClick={() => setActiveTab('ust')}
            className={`px-4 py-1.5 rounded-lg font-medium transition-all whitespace-nowrap flex items-center gap-1.5 ${
              activeTab === 'ust'
                ? 'bg-amber-700 text-white shadow-xs font-semibold'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            <ReceiptText className="w-3.5 h-3.5" />
            Umsatzsteuer
          </button>
        </div>
      </div>

      {/* ------------------------------------------------------------------ */}
      {/* 1. GUV VIEW                                                        */}
      {/* ------------------------------------------------------------------ */}
      {activeTab === 'guv' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Revenue */}
          <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2">
                <TrendingUp className="w-4 h-4 text-amber-700" />
                1. Einnahmen (Erlöse)
              </h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCents(summary?.totalRevenue || 0)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs max-h-96 overflow-y-auto">
              {revenueAccounts.length === 0 ? (
                <div className="py-6 text-center text-stone-400">Keine Einnahmen gebucht</div>
              ) : (
                revenueAccounts.map((acc) => (
                  <div key={acc.number} className="py-2.5 flex justify-between items-center">
                    <div>
                      <span className="font-mono font-bold text-amber-800 mr-2">{acc.number}</span>
                      <span className="text-stone-800">{acc.name}</span>
                    </div>
                    <span className="font-mono font-bold text-stone-900">
                      {formatCents(acc.balance)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Expenses */}
          <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2">
                <TrendingDown className="w-4 h-4 text-rose-600" />
                2. Ausgaben (Aufwendungen)
              </h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCents(summary?.totalExpenses || 0)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs max-h-96 overflow-y-auto">
              {expenseAccounts.length === 0 ? (
                <div className="py-6 text-center text-stone-400">Keine Ausgaben gebucht</div>
              ) : (
                expenseAccounts.map((acc) => (
                  <div key={acc.number} className="py-2.5 flex justify-between items-center">
                    <div>
                      <span className="font-mono font-bold text-rose-800 mr-2">{acc.number}</span>
                      <span className="text-stone-800">{acc.name}</span>
                    </div>
                    <span className="font-mono font-bold text-stone-900">
                      {formatCents(acc.balance)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* GuV Total Net Income Card */}
          <div className="lg:col-span-2 bg-amber-50/60 border border-amber-200/80 rounded-xl p-5 flex items-center justify-between">
            <div>
              <span className="text-xs uppercase tracking-wider font-bold text-amber-900">
                Vorläufiges Jahresergebnis
              </span>
              <p className="text-xs text-amber-800/80 mt-0.5">
                Einnahmen abzüglich Ausgaben vor Steuern
              </p>
            </div>
            <div className="text-2xl font-extrabold font-mono text-amber-900">
              {formatCents(summary?.netIncome || 0)}
            </div>
          </div>
        </div>
      )}

      {/* ------------------------------------------------------------------ */}
      {/* 2. BILANZ VIEW                                                     */}
      {/* ------------------------------------------------------------------ */}
      {activeTab === 'bilanz' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Aktiva */}
          <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900">Vermögen & Bank (Aktiva)</h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCents(totalAssets)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs max-h-96 overflow-y-auto">
              {assetAccounts.map((acc) => (
                <div key={acc.number} className="py-2.5 flex justify-between items-center">
                  <div>
                    <span className="font-mono font-bold text-emerald-800 mr-2">{acc.number}</span>
                    <span className="text-stone-800">{acc.name}</span>
                  </div>
                  <span className="font-mono font-bold text-stone-900">
                    {formatCents(acc.balance)}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Passiva */}
          <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900">Eigenkapital & Verbindlichkeiten (Passiva)</h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCents(totalLiabilities)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs max-h-96 overflow-y-auto">
              {liabilityAccounts.map((acc) => (
                <div key={acc.number} className="py-2.5 flex justify-between items-center">
                  <div>
                    <span className="font-mono font-bold text-sky-800 mr-2">{acc.number}</span>
                    <span className="text-stone-800">{acc.name}</span>
                  </div>
                  <span className="font-mono font-bold text-stone-900">
                    {formatCents(acc.balance)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ------------------------------------------------------------------ */}
      {/* 3. UMSATZSTEUER VIEW (Automatically adapts to Mandantenkonfiguration) */}
      {/* ------------------------------------------------------------------ */}
      {activeTab === 'ust' && (
        <div className="space-y-6">
            <div className="space-y-6">
              {/* Period Selector Header */}
              <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-semibold text-stone-700">
                    {vatPeriod === 'month' ? 'Monatliche Voranmeldung:' : 'Quartals-Voranmeldung:'}
                  </span>
                  <span className="text-xs text-stone-500 font-mono">Geschäftsjahr {currentYear}</span>
                </div>

                {vatPeriod === 'month' ? (
                  <div className="flex items-center gap-1.5 overflow-x-auto text-xs">
                    <select
                      value={selectedMonth}
                      onChange={(e) => setSelectedMonth(Number(e.target.value))}
                      className="px-3 py-1.5 bg-stone-50 border border-stone-200 rounded-lg text-xs font-semibold text-stone-800 focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20"
                    >
                      {monthNames.map((m, idx) => (
                        <option key={idx + 1} value={idx + 1}>
                          {m} {currentYear}
                        </option>
                      ))}
                    </select>
                  </div>
                ) : (
                  <div className="flex gap-1.5">
                    {[1, 2, 3, 4].map((q) => (
                      <button
                        key={q}
                        onClick={() => setSelectedQuarter(q)}
                        className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                          selectedQuarter === q
                            ? 'bg-amber-700 text-white font-semibold shadow-xs'
                            : 'bg-stone-100 text-stone-600 hover:bg-stone-200'
                        }`}
                      >
                        Q{q} {q === 1 ? '(Jan–Mär)' : q === 2 ? '(Apr–Jun)' : q === 3 ? '(Jul–Sep)' : '(Okt–Dez)'}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* KPI Summary Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-1">
                  <span className="text-[11px] uppercase tracking-wider text-stone-500 font-medium">
                    Umsätze (Netto)
                  </span>
                  <div className="text-lg font-bold font-mono text-stone-900">
                    {formatCents(activePeriodVat.totalRevenueNet)}
                  </div>
                  <span className="text-[11px] text-stone-400">Im gewählten Zeitraum</span>
                </div>

                <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-1">
                  <span className="text-[11px] uppercase tracking-wider text-stone-500 font-medium">
                    Umsatzsteuer
                  </span>
                  <div className="text-lg font-bold font-mono text-amber-800">
                    {formatCents(activePeriodVat.totalTax)}
                  </div>
                  <span className="text-[11px] text-stone-400">19% &bull; 7% auf Erlöse</span>
                </div>

                <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-1">
                  <span className="text-[11px] uppercase tracking-wider text-stone-500 font-medium">
                    Abziehbare Vorsteuer
                  </span>
                  <div className="text-lg font-bold font-mono text-emerald-700">
                    {formatCents(activePeriodVat.inputTax)}
                  </div>
                  <span className="text-[11px] text-stone-400">Aus Betriebsausgaben</span>
                </div>

                <div
                  className={`p-4 rounded-xl border shadow-xs space-y-1 ${
                    activePeriodVat.zahllast >= 0
                      ? 'bg-amber-50/70 border-amber-200/80 text-amber-950'
                      : 'bg-emerald-50/70 border-emerald-200 text-emerald-950'
                  }`}
                >
                  <span className="text-[11px] uppercase tracking-wider font-bold">
                    {activePeriodVat.zahllast >= 0 ? 'Verbleibende Zahllast' : 'Erstattungsanspruch'}
                  </span>
                  <div className="text-xl font-extrabold font-mono">
                    {formatCents(Math.abs(activePeriodVat.zahllast))}
                  </div>
                  <span className="text-[11px] opacity-80">
                    {activePeriodVat.zahllast >= 0 ? 'An das Finanzamt zu zahlen' : 'Guthaben vom Finanzamt'}
                  </span>
                </div>
              </div>

              {/* Positions Table (Kennziffern) */}
              <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
                <div className="p-4 border-b border-stone-200 bg-stone-50/60 flex items-center justify-between">
                  <h3 className="text-xs font-bold text-stone-900 uppercase tracking-wider">
                    UStVA-Positionsübersicht (Kennziffern)
                  </h3>
                  <span className="text-xs text-stone-500">
                    {vatPeriod === 'month' ? `Monat ${monthNames[selectedMonth - 1]}` : `Quartal Q${selectedQuarter}`} &bull; {currentYear}
                  </span>
                </div>

                <div className="divide-y divide-stone-100 text-xs">
                  <div className="p-3.5 flex items-center justify-between hover:bg-stone-50/50">
                    <div>
                      <span className="font-mono font-bold text-stone-700 mr-2">Kz 81</span>
                      <span className="text-stone-900 font-medium">
                        Steuerpflichtige Umsätze zum Steuersatz von 19 %
                      </span>
                    </div>
                    <div className="text-right font-mono">
                      <span className="text-stone-600 mr-4">
                        Bemessung: {formatCents(activePeriodVat.rev19Net)}
                      </span>
                      <span className="font-bold text-stone-900">
                        Steuer: {formatCents(activePeriodVat.tax19)}
                      </span>
                    </div>
                  </div>

                  <div className="p-3.5 flex items-center justify-between hover:bg-stone-50/50">
                    <div>
                      <span className="font-mono font-bold text-stone-700 mr-2">Kz 86</span>
                      <span className="text-stone-900 font-medium">
                        Steuerpflichtige Umsätze zum Steuersatz von 7 %
                      </span>
                    </div>
                    <div className="text-right font-mono">
                      <span className="text-stone-600 mr-4">
                        Bemessung: {formatCents(activePeriodVat.rev7Net)}
                      </span>
                      <span className="font-bold text-stone-900">
                        Steuer: {formatCents(activePeriodVat.tax7)}
                      </span>
                    </div>
                  </div>

                  <div className="p-3.5 flex items-center justify-between hover:bg-stone-50/50">
                    <div>
                      <span className="font-mono font-bold text-stone-700 mr-2">Kz 66</span>
                      <span className="text-stone-900 font-medium">
                        Abziehbare Vorsteuerbeträge (aus Rechnungen anderer Unternehmen)
                      </span>
                    </div>
                    <div className="text-right font-mono font-bold text-emerald-700">
                      - {formatCents(activePeriodVat.inputTax)}
                    </div>
                  </div>

                  <div className="p-4 bg-stone-50 flex items-center justify-between font-bold text-sm">
                    <div>
                      <span className="font-mono text-amber-800 mr-2">Kz 83</span>
                      <span className="text-stone-900">
                        Verbleibende Umsatzsteuer-Vorauszahlung
                      </span>
                    </div>
                    <div
                      className={`font-mono text-base ${
                        activePeriodVat.zahllast >= 0 ? 'text-amber-900' : 'text-emerald-700'
                      }`}
                    >
                      {formatCents(activePeriodVat.zahllast)}
                    </div>
                  </div>
                </div>
              </div>

              {/* Quarterly Comparison Table */}
              {vatPeriod === 'quarter' && (
                <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
                  <div className="p-4 border-b border-stone-200 bg-stone-50/60 flex items-center justify-between">
                    <h3 className="text-xs font-bold text-stone-900 uppercase tracking-wider">
                      Jahresverlauf aller 4 Quartale
                    </h3>
                    <span className="text-xs text-stone-500">Geschäftsjahr {currentYear}</span>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
                        <tr>
                          <th className="py-2.5 px-4">Zeitraum</th>
                          <th className="py-2.5 px-4 text-right">Umsatz Netto</th>
                          <th className="py-2.5 px-4 text-right">Umsatzsteuer</th>
                          <th className="py-2.5 px-4 text-right">Vorsteuer</th>
                          <th className="py-2.5 px-4 text-right">Zahllast / Erstattung</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-stone-100 font-mono">
                        {quartersData.map((q) => (
                          <tr key={q.quarter} className="hover:bg-stone-50/50">
                            <td className="py-3 px-4 font-sans font-medium text-stone-900">
                              {q.label}
                            </td>
                            <td className="py-3 px-4 text-right text-stone-700">
                              {formatCents(q.totalRevenueNet)}
                            </td>
                            <td className="py-3 px-4 text-right text-amber-800">
                              {formatCents(q.totalTax)}
                            </td>
                            <td className="py-3 px-4 text-right text-emerald-700">
                              {formatCents(q.inputTax)}
                            </td>
                            <td
                              className={`py-3 px-4 text-right font-bold ${
                                q.zahllast >= 0 ? 'text-stone-900' : 'text-emerald-700'
                              }`}
                            >
                              {formatCents(q.zahllast)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
        </div>
      )}
    </div>
  );
};

/** Letzter Tag eines Monats als ISO-Datum. */
function endOfMonth(year: number, month: number): string {
  const last = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return `${year}-${String(month).padStart(2, '0')}-${String(last).padStart(2, '0')}`;
}
