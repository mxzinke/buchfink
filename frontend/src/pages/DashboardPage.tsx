import React, { useEffect, useState } from 'react';
import {
  TrendingUp,
  TrendingDown,
  Wallet,
  ArrowUpRight,
  ArrowDownRight,
  FileText,
  Landmark,
  Sparkles,
} from 'lucide-react';
import { FinancialSummary, BookingEntry } from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

interface DashboardPageProps {
  onNavigate: (tab: any) => void;
}

export const DashboardPage: React.FC<DashboardPageProps> = ({ onNavigate }) => {
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [recentBookings, setRecentBookings] = useState<BookingEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [sum, bookings] = await Promise.all([
        Api.getFinancialSummary(),
        Api.getBookings(),
      ]);
      setSummary(sum);
      setRecentBookings(bookings.slice(-8).reverse());
    } finally {
      setLoading(false);
    }
  };

  const handleSeedSampleData = async () => {
    setLoading(true);
    try {
      await Api.importSampleBankStatement();
      onNavigate('bank');
    } finally {
      setLoading(false);
    }
  };

  if (loading || !summary) {
    return (
      <div className="p-8 flex items-center justify-center text-stone-500 text-xs">
        Kennzahlen werden geladen...
      </div>
    );
  }

  const hasData = recentBookings.length > 0 || summary.totalRevenue > 0 || summary.totalExpenses > 0;

  return (
    <div className="p-8 max-w-6xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight">
            Buchhaltungsübersicht
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            GoBD-konform &bull; 100% Local-First
          </p>
        </div>
        <div className="flex gap-2.5">
          <button
            onClick={() => onNavigate('bank')}
            className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-stone-900 text-stone-100 text-xs font-semibold hover:bg-stone-800 transition-colors shadow-xs"
          >
            <Landmark className="w-3.5 h-3.5 text-amber-400" />
            Bankumsätze abgleichen
          </button>
          <button
            onClick={() => onNavigate('invoices')}
            className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-amber-600 text-white text-xs font-semibold hover:bg-amber-700 transition-colors shadow-xs"
          >
            <FileText className="w-3.5 h-3.5" />
            Neue Rechnung
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Bank */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/90 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Bankguthaben
              <HelpTooltip
                title="Liquide Mittel (Konto 1800)"
                content="Aktueller Gesamtsaldo der Geschäftskonten nach Verbuchung."
              />
            </span>
            <div className="p-1.5 rounded-lg bg-emerald-50 text-emerald-700">
              <Wallet className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-stone-900">
            {formatCurrency(summary.bankBalance)}
          </div>
          <div className="text-[11px] text-stone-500 mt-1">Konto 1800 (Bank)</div>
        </div>

        {/* Revenue */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/90 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Umsatzerlöse
              <HelpTooltip
                title="Erlöskonten (Klasse 4)"
                content="Summe aller Erträge im laufenden Geschäftsjahr (z. B. Konto 4400 Erlöse 19% USt)."
              />
            </span>
            <div className="p-1.5 rounded-lg bg-amber-50 text-amber-700">
              <TrendingUp className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-stone-900">
            {formatCurrency(summary.totalRevenue)}
          </div>
          <div className="text-[11px] text-stone-500 mt-1">Erträge (GuV)</div>
        </div>

        {/* Expenses */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/90 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Gesamtaufwand
              <HelpTooltip
                title="Aufwandskonten (Klasse 6)"
                content="Betriebsausgaben wie Miete, Software, Fremdleistungen etc."
              />
            </span>
            <div className="p-1.5 rounded-lg bg-rose-50 text-rose-700">
              <TrendingDown className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-stone-900">
            {formatCurrency(summary.totalExpenses)}
          </div>
          <div className="text-[11px] text-stone-500 mt-1">Ausgaben (GuV)</div>
        </div>

        {/* Net Income */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/90 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Vorl. Ergebnis
              <HelpTooltip
                title="Vorläufiger Gewinn/Verlust"
                content="Differenz aus Umsatzerlösen und Betriebsausgaben vor Steuern."
              />
            </span>
            <div
              className={`p-1.5 rounded-lg ${
                summary.netIncome >= 0 ? 'bg-amber-50 text-amber-700' : 'bg-rose-50 text-rose-700'
              }`}
            >
              {summary.netIncome >= 0 ? (
                <ArrowUpRight className="w-4 h-4" />
              ) : (
                <ArrowDownRight className="w-4 h-4" />
              )}
            </div>
          </div>
          <div
            className={`text-xl font-bold ${
              summary.netIncome >= 0 ? 'text-amber-800' : 'text-rose-700'
            }`}
          >
            {formatCurrency(summary.netIncome)}
          </div>
          <div className="text-[11px] text-stone-500 mt-1">GuV-Saldo</div>
        </div>
      </div>

      {/* Empty State vs Recent Bookings */}
      {!hasData ? (
        <div className="bg-white rounded-2xl border border-stone-200/90 p-12 text-center space-y-4 shadow-xs">
          <div className="w-12 h-12 rounded-2xl bg-amber-50 text-amber-600 flex items-center justify-center mx-auto border border-amber-200/60">
            <Landmark className="w-6 h-6" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-bold text-stone-900">Noch keine Buchungen im Journal</h3>
            <p className="text-xs text-stone-500 max-w-md mx-auto">
              Starten Sie mit dem Import eines CAMT.053 Kontoauszugs oder laden Sie den Muster-Auszug, um den automatisierten Buchungsabgleich zu testen.
            </p>
          </div>
          <div className="flex items-center justify-center gap-3 pt-2">
            <button
              onClick={handleSeedSampleData}
              className="px-4 py-2 rounded-lg bg-amber-600 text-white text-xs font-semibold hover:bg-amber-700 transition-colors shadow-xs flex items-center gap-1.5"
            >
              <Sparkles className="w-3.5 h-3.5" /> Muster-Kontoauszug laden
            </button>
            <button
              onClick={() => onNavigate('bank')}
              className="px-4 py-2 rounded-lg bg-stone-100 text-stone-700 text-xs font-semibold hover:bg-stone-200 transition-colors border border-stone-200"
            >
              CAMT.053 XML importieren
            </button>
          </div>
        </div>
      ) : (
        /* Recent Bookings Table */
        <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs overflow-hidden">
          <div className="p-4 border-b border-stone-200/80 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-stone-900">Letzte Journal-Buchungen</h3>
              <p className="text-xs text-stone-500">
                Lückenlose GoBD-Verkettung
              </p>
            </div>
            <button
              onClick={() => onNavigate('journal')}
              className="text-xs font-semibold text-amber-700 hover:text-amber-800"
            >
              Zum Journal &rarr;
            </button>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-stone-50/80 border-b border-stone-200 text-stone-500 font-medium">
                <tr>
                  <th className="py-2.5 px-4">Journal-Nr.</th>
                  <th className="py-2.5 px-4">Datum</th>
                  <th className="py-2.5 px-4">Buchungstext</th>
                  <th className="py-2.5 px-4">Soll &rarr; Haben</th>
                  <th className="py-2.5 px-4 text-right">Betrag</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-stone-100">
                {recentBookings.map((b) => (
                  <tr key={b.id} className="hover:bg-stone-50/50 transition-colors">
                    <td className="py-3 px-4 font-mono font-medium text-amber-800">
                      {b.bookingNumber}
                    </td>
                    <td className="py-3 px-4 text-stone-600 font-mono">{formatDate(b.date)}</td>
                    <td className="py-3 px-4 text-stone-900 font-medium">{b.description}</td>
                    <td className="py-3 px-4 font-mono text-stone-700">
                      {b.debitAccount} &rarr; {b.creditAccount}
                    </td>
                    <td className="py-3 px-4 text-right font-bold font-mono text-stone-900">
                      {formatCurrency(b.amount, b.currency)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
