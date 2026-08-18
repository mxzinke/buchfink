import React, { useEffect, useState } from 'react';
import { Search, Filter } from 'lucide-react';
import { Account } from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatPercent } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const AccountsPage: React.FC = () => {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadAccounts();
  }, []);

  const loadAccounts = async () => {
    setLoading(true);
    try {
      const list = await Api.getAccounts();
      setAccounts(list);
    } finally {
      setLoading(false);
    }
  };

  const filteredAccounts = accounts.filter((acc) => {
    const matchesSearch =
      acc.number.toLowerCase().includes(search.toLowerCase()) ||
      acc.name.toLowerCase().includes(search.toLowerCase()) ||
      acc.category.toLowerCase().includes(search.toLowerCase()) ||
      acc.description.toLowerCase().includes(search.toLowerCase());

    const matchesType = typeFilter === 'all' || acc.type === typeFilter;
    return matchesSearch && matchesType;
  });

  const getTypeBadge = (type: string) => {
    switch (type) {
      case 'asset':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-100 text-emerald-800">Aktiva (Vermögen)</span>;
      case 'liability':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-sky-100 text-sky-800">Passiva (Kapital/Schulden)</span>;
      case 'revenue':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-amber-100 text-amber-800">Ertrag (Erlöse)</span>;
      case 'expense':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-rose-100 text-rose-800">Aufwand (Kosten)</span>;
      default:
        return null;
    }
  };

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Kontenplan
            <HelpTooltip
              title="Deutscher Kontenrahmen (SKR04)"
              content="Aufgebaut nach dem Abschlussgliederungsprinzip (Bilanz & GuV): Klasse 0: Anlagevermögen, Klasse 1: Umlaufvermögen, Klasse 2: Eigenkapital, Klasse 3: Verbindlichkeiten, Klasse 4: Erträge, Klasse 6: Aufwendungen."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Vorinstallierter deutscher Kontenplan mit integrierten Erklärungen für Laien
          </p>
        </div>
      </div>

      <div className="bg-white p-4 rounded-xl border border-stone-200/90 shadow-xs flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
        <div className="relative flex-1">
          <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Konto nach Nummer (z. B. 1800, 4400) oder Name/Stichwort suchen..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
          />
        </div>

        <div className="flex items-center gap-1.5 overflow-x-auto text-xs">
          <span className="text-stone-400 flex items-center gap-1 text-[11px] mr-1">
            <Filter className="w-3 h-3" /> Filter:
          </span>
          {[
            { id: 'all', label: 'Alle Konten' },
            { id: 'asset', label: 'Aktiva (Klasse 0-1)' },
            { id: 'liability', label: 'Passiva (Klasse 2-3)' },
            { id: 'revenue', label: 'Erträge (Klasse 4)' },
            { id: 'expense', label: 'Aufwand (Klasse 6)' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setTypeFilter(tab.id)}
              className={`px-3 py-1.5 rounded-lg font-medium whitespace-nowrap transition-colors ${
                typeFilter === tab.id
                  ? 'bg-amber-600 text-white shadow-xs'
                  : 'bg-stone-100 text-stone-600 hover:bg-stone-200/70'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
              <tr>
                <th className="py-3 px-4 w-24">Kontonr.</th>
                <th className="py-3 px-4">Kontenbezeichnung & Erläuterung</th>
                <th className="py-3 px-4">Kategorie</th>
                <th className="py-3 px-4">Art</th>
                <th className="py-3 px-4 text-center">Steuersatz</th>
                <th className="py-3 px-4 text-right">Aktueller Saldo</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100">
              {loading ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-stone-400">
                    Kontenplan wird geladen...
                  </td>
                </tr>
              ) : filteredAccounts.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-stone-400">
                    Keine Konten gefunden für "{search}".
                  </td>
                </tr>
              ) : (
                filteredAccounts.map((acc) => (
                  <tr key={acc.number} className="hover:bg-amber-50/30 transition-colors">
                    <td className="py-3 px-4 font-mono font-bold text-amber-800">
                      {acc.number}
                    </td>
                    <td className="py-3 px-4">
                      <div className="font-semibold text-stone-900">{acc.name}</div>
                      {acc.description && (
                        <div className="text-[11px] text-stone-500 mt-0.5 leading-snug">
                          {acc.description}
                        </div>
                      )}
                    </td>
                    <td className="py-3 px-4 text-stone-600">{acc.category}</td>
                    <td className="py-3 px-4">{getTypeBadge(acc.type)}</td>
                    <td className="py-3 px-4 text-center font-mono text-stone-600">
                      {acc.taxRate > 0 ? formatPercent(acc.taxRate) : '—'}
                    </td>
                    <td className="py-3 px-4 text-right font-mono font-bold text-stone-900">
                      {formatCurrency(acc.balance)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
