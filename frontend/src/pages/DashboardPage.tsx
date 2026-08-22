// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

import React, { useEffect, useState } from 'react';
import {
  TrendingUp,
  TrendingDown,
  Wallet,
  ArrowUpRight,
  ArrowDownRight,
  FileText,
  Landmark,
} from 'lucide-react';
import { FinancialSummary, JournalEntry, CompanySettings } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

interface DashboardPageProps {
  onNavigate: (tab: any) => void;
}

export const DashboardPage: React.FC<DashboardPageProps> = ({ onNavigate }) => {
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [recentEntries, setRecentEntries] = useState<JournalEntry[]>([]);
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
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

  if (loading || !summary) {
    return (
      <div className="p-8 flex items-center justify-center text-stone-500 text-xs">
        Kennzahlen werden geladen...
      </div>
    );
  }

  const hasData = recentEntries.length > 0 || summary.totalRevenue > 0 || summary.totalExpenses > 0;

  return (
    <div className="p-8 max-w-6xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-2xl font-bold text-stone-900 tracking-tight">
              Buchhaltungsübersicht
            </h2>
            {settings?.isSmallBusiness && (
              <span className="px-2 py-0.5 rounded-md text-[11px] font-semibold bg-amber-100 text-amber-800 border border-amber-200">
                § 19 UStG (Kleinunternehmer)
              </span>
            )}
            {settings?.taxationType && (
              <span className="px-2 py-0.5 rounded-md text-[11px] font-medium bg-stone-100 text-stone-600 border border-stone-200">
                {settings.taxationType}-Versteuerung
              </span>
            )}
          </div>
          <p className="text-xs text-stone-500 mt-1">
            Lokal gespeichert &bull; Rechtssicher nach deutschem Standard
          </p>
        </div>
        <div className="flex gap-2.5">
          <button
            onClick={() => onNavigate('bank')}
            className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-stone-700 text-stone-100 text-xs font-semibold hover:bg-stone-800 transition-colors shadow-xs"
          >
            <Landmark className="w-3.5 h-3.5 text-amber-300" />
            Bankumsätze abgleichen
          </button>
          <button
            onClick={() => onNavigate('invoices')}
            className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs"
          >
            <FileText className="w-3.5 h-3.5" />
            Neue Rechnung
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Bank */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/80 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Bankguthaben
              <HelpTooltip
                title="Bankguthaben"
                content="Aktueller Gesamtsaldo auf Ihrem Geschäftskonto."
              />
            </span>
            <div className="p-1.5 rounded-lg bg-emerald-50 text-emerald-700">
              <Wallet className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-stone-900">
            {formatCents(summary.bankBalance)}
          </div>
          <div className="text-xs text-stone-500 mt-1">Geschäftskonto (1800)</div>
        </div>

        {/* Revenue */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/80 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Einnahmen
              <HelpTooltip
                title="Einnahmen"
                content="Summe aller Erlöse im laufenden Geschäftsjahr vor Steuern."
              />
            </span>
            <div className="p-1.5 rounded-lg bg-amber-50 text-amber-700">
              <TrendingUp className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-stone-900">
            {formatCents(summary.totalRevenue)}
          </div>
          <div className="text-xs text-stone-500 mt-1">Gesamterlöse</div>
        </div>

        {/* Expenses */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/80 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Ausgaben
              <HelpTooltip
                title="Ausgaben"
                content="Summe aller laufenden Betriebsausgaben wie Miete, Software und Dienstleistungen."
              />
            </span>
            <div className="p-1.5 rounded-lg bg-rose-50 text-rose-700">
              <TrendingDown className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold text-stone-900">
            {formatCents(summary.totalExpenses)}
          </div>
          <div className="text-xs text-stone-500 mt-1">Betriebsausgaben</div>
        </div>

        {/* Net Income */}
        <div className="bg-white p-5 rounded-xl border border-stone-200/80 shadow-xs">
          <div className="flex items-center justify-between text-stone-500 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider flex items-center">
              Ergebnis
              <HelpTooltip
                title="Vorläufiges Ergebnis"
                content="Gewinn bzw. Verlust vor Steuern (Einnahmen minus Ausgaben)."
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
            {formatCents(summary.netIncome)}
          </div>
          <div className="text-xs text-stone-500 mt-1">Einnahmen minus Ausgaben</div>
        </div>
      </div>

      {/* Empty State vs Recent Bookings */}
      {!hasData ? (
        <div className="bg-white rounded-2xl border border-stone-200/80 p-12 text-center space-y-4 shadow-xs">
          <div className="w-12 h-12 rounded-2xl bg-amber-50 text-amber-700 flex items-center justify-center mx-auto border border-amber-200/60">
            <Landmark className="w-6 h-6" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-bold text-stone-900">Noch keine Buchungen erfasst</h3>
            <p className="text-xs text-stone-500 max-w-md mx-auto">
              Starten Sie mit dem Import eines Kontoauszugs oder erfassen Sie Buchungen direkt im Journal.
            </p>
          </div>
          <div className="flex items-center justify-center gap-3 pt-2">
            <button
              onClick={() => onNavigate('bank')}
              className="px-4 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs flex items-center gap-1.5"
            >
              <Landmark className="w-3.5 h-3.5" /> Kontoauszug importieren
            </button>
            <button
              onClick={() => onNavigate('journal')}
              className="px-4 py-2 rounded-lg bg-stone-100 text-stone-700 text-xs font-semibold hover:bg-stone-200 transition-colors border border-stone-200"
            >
              Zum Journal
            </button>
          </div>
        </div>
      ) : (
        /* Recent Bookings Table */
        <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
          <div className="p-4 border-b border-stone-200/80 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-stone-900">Letzte Buchungen</h3>
              <p className="text-xs text-stone-500">
                Laufend erfasst und dokumentiert
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
                  <th className="py-2.5 px-4">Nr.</th>
                  <th className="py-2.5 px-4">Datum</th>
                  <th className="py-2.5 px-4">Buchungstext</th>
                  <th className="py-2.5 px-4">Konten (Soll &rarr; Haben)</th>
                  <th className="py-2.5 px-4 text-right">Betrag</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-stone-100">
                {recentEntries.map((entry) => {
                  const gross = entry.lines
                    .filter((l) => l.side === 'S')
                    .reduce((sum, l) => sum + l.amount, 0);
                  const isReversal = entry.kind === 'reversal';

                  return (
                    <tr
                      key={entry.id}
                      className={`transition-colors ${
                        isReversal ? 'bg-rose-50/30 text-stone-600' : 'hover:bg-stone-50/50'
                      }`}
                    >
                      <td className="py-3 px-4 font-mono font-medium text-amber-800">
                        {entry.entryNumber}
                      </td>
                      <td className="py-3 px-4 text-stone-600">{formatDate(entry.bookingDate)}</td>
                      <td className="py-3 px-4 text-stone-900 font-medium">
                        {entry.description}
                        {isReversal && (
                          <span className="ml-2 text-[10px] text-rose-600 font-sans font-normal">
                            (Generalumkehr)
                          </span>
                        )}
                      </td>
                      <td className="py-3 px-4 font-mono text-xs text-stone-700">
                        {entry.lines.map((l) => l.account).join(' · ')}
                      </td>
                      <td className="py-3 px-4 text-right font-semibold font-mono text-stone-900">
                        {formatCents(gross, entry.currency)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
