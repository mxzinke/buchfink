import React, { useEffect, useState } from 'react';
import { TrendingUp, TrendingDown } from 'lucide-react';
import { Account, FinancialSummary } from '../types';
import { Api } from '../services/api';
import { formatCurrency } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const ReportsPage: React.FC = () => {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [summary, setSummary] = useState<FinancialSummary | null>(null);
  const [activeTab, setActiveTab] = useState<'guv' | 'bilanz'>('guv');

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [accs, sum] = await Promise.all([
        Api.getAccounts(),
        Api.getFinancialSummary(),
      ]);
      setAccounts(accs);
      setSummary(sum);
    } catch (e) {
      console.error(e);
    }
  };

  const revenueAccounts = accounts.filter((a) => a.type === 'revenue' && a.balance !== 0);
  const expenseAccounts = accounts.filter((a) => a.type === 'expense' && a.balance !== 0);
  const assetAccounts = accounts.filter((a) => a.type === 'asset' && a.balance !== 0);
  const liabilityAccounts = accounts.filter((a) => a.type === 'liability' && a.balance !== 0);

  const totalAssets = assetAccounts.reduce((sum, a) => sum + a.balance, 0);
  const totalLiabilities = liabilityAccounts.reduce((sum, a) => sum + a.balance, 0);

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Auswertungen (GuV & Bilanz)
            <HelpTooltip
              title="GuV & Bilanzierung nach SKR04"
              content="Die Gewinn- und Verlustrechnung (GuV) stellt Erträge und Aufwendungen gegenüber. Die Bilanz zeigt Aktiva (Vermögensverwendung) und Passiva (Mittelherkunft / Eigenkapital & Verbindlichkeiten)."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Echtzeit-Berechnung aus dem Journal ohne manuelle Monatsabschlüsse
          </p>
        </div>

        {/* View Switcher */}
        <div className="flex bg-stone-100 p-1 rounded-lg border border-stone-200 text-xs">
          <button
            onClick={() => setActiveTab('guv')}
            className={`px-4 py-1.5 rounded-md font-semibold transition-all ${
              activeTab === 'guv'
                ? 'bg-amber-600 text-white shadow-xs'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Gewinn- & Verlustrechnung (GuV)
          </button>
          <button
            onClick={() => setActiveTab('bilanz')}
            className={`px-4 py-1.5 rounded-md font-semibold transition-all ${
              activeTab === 'bilanz'
                ? 'bg-amber-600 text-white shadow-xs'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Bilanz (Aktiva / Passiva)
          </button>
        </div>
      </div>

      {activeTab === 'guv' ? (
        /* GuV View */
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Revenue */}
          <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2">
                <TrendingUp className="w-4 h-4 text-amber-600" />
                1. Betriebliche Erträge (Klasse 4)
              </h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCurrency(summary?.totalRevenue || 0)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs">
              {revenueAccounts.length === 0 ? (
                <div className="py-6 text-center text-stone-400">Keine Erträge gebucht</div>
              ) : (
                revenueAccounts.map((acc) => (
                  <div key={acc.number} className="py-2.5 flex justify-between items-center">
                    <div>
                      <span className="font-mono font-bold text-amber-800 mr-2">{acc.number}</span>
                      <span className="text-stone-800">{acc.name}</span>
                    </div>
                    <span className="font-mono font-bold text-stone-900">
                      {formatCurrency(acc.balance)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Expenses */}
          <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2">
                <TrendingDown className="w-4 h-4 text-rose-600" />
                2. Betriebliche Aufwendungen (Klasse 6)
              </h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCurrency(summary?.totalExpenses || 0)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs">
              {expenseAccounts.length === 0 ? (
                <div className="py-6 text-center text-stone-400">Keine Aufwendungen gebucht</div>
              ) : (
                expenseAccounts.map((acc) => (
                  <div key={acc.number} className="py-2.5 flex justify-between items-center">
                    <div>
                      <span className="font-mono font-bold text-rose-800 mr-2">{acc.number}</span>
                      <span className="text-stone-800">{acc.name}</span>
                    </div>
                    <span className="font-mono font-bold text-stone-900">
                      {formatCurrency(acc.balance)}
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
                Vorläufiges Jahresergebnis (GuV Saldo)
              </span>
              <p className="text-[11px] text-amber-800/80 mt-0.5">
                Erträge abzüglich Aufwendungen vor Steuern
              </p>
            </div>
            <div className="text-2xl font-extrabold font-mono text-amber-900">
              {formatCurrency(summary?.netIncome || 0)}
            </div>
          </div>
        </div>
      ) : (
        /* Bilanz View */
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Aktiva */}
          <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900">AKTIVA (Mittelverwendung)</h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCurrency(totalAssets)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs">
              {assetAccounts.map((acc) => (
                <div key={acc.number} className="py-2.5 flex justify-between items-center">
                  <div>
                    <span className="font-mono font-bold text-emerald-800 mr-2">{acc.number}</span>
                    <span className="text-stone-800">{acc.name}</span>
                  </div>
                  <span className="font-mono font-bold text-stone-900">
                    {formatCurrency(acc.balance)}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Passiva */}
          <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-stone-100 pb-3">
              <h3 className="text-sm font-bold text-stone-900">PASSIVA (Mittelherkunft)</h3>
              <span className="font-mono font-bold text-sm text-stone-900">
                {formatCurrency(totalLiabilities)}
              </span>
            </div>

            <div className="divide-y divide-stone-100 text-xs">
              {liabilityAccounts.map((acc) => (
                <div key={acc.number} className="py-2.5 flex justify-between items-center">
                  <div>
                    <span className="font-mono font-bold text-sky-800 mr-2">{acc.number}</span>
                    <span className="text-stone-800">{acc.name}</span>
                  </div>
                  <span className="font-mono font-bold text-stone-900">
                    {formatCurrency(acc.balance)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
