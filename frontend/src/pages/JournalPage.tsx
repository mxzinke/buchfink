import React, { useEffect, useState } from 'react';
import {
  ShieldCheck,
  RotateCcw,
  Search,
  FileCode,
  Lock,
} from 'lucide-react';
import { BookingEntry } from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatDate, formatShortHash } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const JournalPage: React.FC = () => {
  const [bookings, setBookings] = useState<BookingEntry[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [stornoModalBooking, setStornoModalBooking] = useState<BookingEntry | null>(null);
  const [stornoReason, setStornoReason] = useState('');
  const [isSubmittingStorno, setIsSubmittingStorno] = useState(false);

  useEffect(() => {
    loadBookings();
  }, []);

  const loadBookings = async () => {
    setLoading(true);
    try {
      const list = await Api.getBookings();
      setBookings(list);
    } finally {
      setLoading(false);
    }
  };

  const handleStorno = async () => {
    if (!stornoModalBooking || !stornoReason.trim()) return;
    setIsSubmittingStorno(true);
    try {
      await Api.stornoBooking(stornoModalBooking.id, stornoReason);
      setStornoModalBooking(null);
      setStornoReason('');
      await loadBookings();
    } finally {
      setIsSubmittingStorno(false);
    }
  };

  const filteredBookings = bookings.filter((b) => {
    return (
      b.bookingNumber.toLowerCase().includes(search.toLowerCase()) ||
      b.description.toLowerCase().includes(search.toLowerCase()) ||
      b.debitAccount.includes(search) ||
      b.creditAccount.includes(search) ||
      b.receiptNumber.toLowerCase().includes(search.toLowerCase())
    );
  });

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Buchungsjournal
            <HelpTooltip
              title="Doppelte Buchführung & Journal"
              content="Jede Buchung berührt immer mindestens zwei Konten (Soll an Haben). Buchungen entstehen in Buchfink automatisiert aus Bankzuordnungen und Belegen. Korrekturen erfolgen nach GoBD ausschließlich per Storno."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Kryptografisch verkettete Buchungssätze &bull; Unveränderbar & GoBD-konform
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-stone-100 text-stone-700 text-xs font-mono border border-stone-200">
            <Lock className="w-3.5 h-3.5 text-stone-500" />
            <span>Kryptografische Hash-Chain aktiv</span>
          </div>
        </div>
      </div>

      {/* Search */}
      <div className="bg-white p-4 rounded-xl border border-stone-200/90 shadow-xs">
        <div className="relative">
          <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Journal durchsuchen nach Beleg-Nr., Text oder Kontonummer..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
          />
        </div>
      </div>

      {/* Bookings Table */}
      <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
              <tr>
                <th className="py-3 px-4 w-28">Journal-Nr.</th>
                <th className="py-3 px-4 w-24">Datum</th>
                <th className="py-3 px-4">Buchungstext & Beleg</th>
                <th className="py-3 px-4">Soll-Konto</th>
                <th className="py-3 px-4">Haben-Konto</th>
                <th className="py-3 px-4 text-right">Betrag</th>
                <th className="py-3 px-4 text-center">Hash-Kette</th>
                <th className="py-3 px-4 text-right">Aktion</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100">
              {loading ? (
                <tr>
                  <td colSpan={8} className="py-8 text-center text-stone-400">
                    Journal wird geladen...
                  </td>
                </tr>
              ) : filteredBookings.length === 0 ? (
                <tr>
                  <td colSpan={8} className="py-8 text-center text-stone-400">
                    Keine Buchungen gefunden.
                  </td>
                </tr>
              ) : (
                filteredBookings.map((b) => (
                  <tr
                    key={b.id}
                    className={`hover:bg-amber-50/20 transition-colors ${
                      b.isStorno ? 'bg-rose-50/30 line-through text-stone-400' : ''
                    }`}
                  >
                    <td className="py-3 px-4 font-mono font-bold text-stone-800">
                      {b.bookingNumber}
                    </td>
                    <td className="py-3 px-4 text-stone-600 font-mono">
                      {formatDate(b.date)}
                    </td>
                    <td className="py-3 px-4">
                      <div className="font-semibold text-stone-900">{b.description}</div>
                      {b.receiptNumber && (
                        <div className="text-[11px] text-amber-800 font-mono mt-0.5 flex items-center gap-1">
                          <FileCode className="w-3 h-3 text-amber-600" />
                          <span>Beleg: {b.receiptNumber}</span>
                          {b.receiptHash && (
                            <span className="text-stone-400 text-[10px]">
                              ({formatShortHash(b.receiptHash)})
                            </span>
                          )}
                        </div>
                      )}
                    </td>
                    <td className="py-3 px-4 font-mono">
                      <div className="font-bold text-stone-900">{b.debitAccount}</div>
                      <div className="text-[10px] text-stone-500 font-sans truncate max-w-[140px]">
                        {b.debitAccountName}
                      </div>
                    </td>
                    <td className="py-3 px-4 font-mono">
                      <div className="font-bold text-stone-900">{b.creditAccount}</div>
                      <div className="text-[10px] text-stone-500 font-sans truncate max-w-[140px]">
                        {b.creditAccountName}
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right font-mono font-bold text-stone-900">
                      {formatCurrency(b.amount, b.currency)}
                    </td>
                    <td className="py-3 px-4 text-center">
                      <div
                        className="inline-flex items-center gap-1 text-[10px] font-mono text-stone-500 bg-stone-100 px-2 py-0.5 rounded cursor-help"
                        title={`Prev: ${b.previousHash}\nHash: ${b.entryHash}`}
                      >
                        <ShieldCheck className="w-3 h-3 text-emerald-600" />
                        <span>{formatShortHash(b.entryHash)}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right">
                      {!b.isStorno && (
                        <button
                          onClick={() => setStornoModalBooking(b)}
                          className="px-2.5 py-1 text-[11px] font-medium text-rose-700 bg-rose-50 hover:bg-rose-100 rounded-md transition-colors inline-flex items-center gap-1"
                          title="GoBD-konform stornieren"
                        >
                          <RotateCcw className="w-3 h-3" />
                          Storno
                        </button>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Storno Modal Dialog */}
      {stornoModalBooking && (
        <div className="fixed inset-0 z-50 bg-stone-900/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-xl shadow-2xl max-w-md w-full p-6 border border-stone-200 animate-in fade-in zoom-in-95 duration-150">
            <h3 className="text-base font-bold text-stone-900 flex items-center gap-2">
              <RotateCcw className="w-4 h-4 text-rose-600" />
              Buchung stornieren (GoBD)
            </h3>
            <p className="text-xs text-stone-500 mt-1">
              Nach den GoBD können festgeschriebene Buchungen nicht gelöscht werden. Es wird eine
              Gegenbuchung mit neuer Hash-Kette erzeugt.
            </p>

            <div className="my-4 p-3 bg-stone-50 rounded-lg text-xs space-y-1 font-mono">
              <div className="text-stone-600">
                Buchung: <span className="font-bold text-stone-900">{stornoModalBooking.bookingNumber}</span>
              </div>
              <div className="text-stone-600">
                Text: <span className="text-stone-900 font-sans">{stornoModalBooking.description}</span>
              </div>
              <div className="text-stone-600">
                Betrag: <span className="font-bold text-stone-900">{formatCurrency(stornoModalBooking.amount)}</span>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-xs font-semibold text-stone-700 block">
                Stornogrund (Pflichtfeld für Verfahrensdokumentation):
              </label>
              <textarea
                rows={3}
                placeholder="z. B. Falsches Aufwandskonto gewählt, Rechnungskorrektur durch Lieferant..."
                value={stornoReason}
                onChange={(e) => setStornoReason(e.target.value)}
                className="w-full text-xs p-2.5 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-rose-500/20 focus:border-rose-500"
              />
            </div>

            <div className="flex justify-end gap-2 mt-6">
              <button
                type="button"
                onClick={() => setStornoModalBooking(null)}
                className="px-3.5 py-1.5 text-xs font-medium text-stone-600 hover:bg-stone-100 rounded-lg"
              >
                Abbrechen
              </button>
              <button
                type="button"
                disabled={!stornoReason.trim() || isSubmittingStorno}
                onClick={handleStorno}
                className="px-3.5 py-1.5 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-700 disabled:opacity-50 rounded-lg shadow-xs"
              >
                {isSubmittingStorno ? 'Wird gebucht...' : 'Stornobuchung erzeugen'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
