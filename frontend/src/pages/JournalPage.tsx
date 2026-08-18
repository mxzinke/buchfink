import React, { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import {
  ShieldCheck,
  RotateCcw,
  Search,
  FileCode,
  Lock,
  Plus,
  AlertTriangle,
  CheckCircle2,
} from 'lucide-react';
import { BookingEntry, Account, CompanySettings } from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const JournalPage: React.FC = () => {
  const [bookings, setBookings] = useState<BookingEntry[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);

  // Storno Modal
  const [stornoModalBooking, setStornoModalBooking] = useState<BookingEntry | null>(null);
  const [stornoReason, setStornoReason] = useState('');
  const [stornoError, setStornoError] = useState<string | null>(null);
  const [isSubmittingStorno, setIsSubmittingStorno] = useState(false);

  // New Booking Modal
  const [isNewBookingModalOpen, setIsNewBookingModalOpen] = useState(false);
  const [bookingDate, setBookingDate] = useState(new Date().toISOString().split('T')[0]);
  const [bookingDesc, setBookingDesc] = useState('');
  const [debitAccount, setDebitAccount] = useState('1800');
  const [creditAccount, setCreditAccount] = useState('4400');
  const [amount, setAmount] = useState<number | ''>('');
  const [receiptNumber, setReceiptNumber] = useState('');
  const [taxCode, setTaxCode] = useState('NONE');
  const [assignedFiscalYear, setAssignedFiscalYear] = useState<number>(new Date().getFullYear());
  const [isSubmittingBooking, setIsSubmittingBooking] = useState(false);
  const [newBookingError, setNewBookingError] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [list, accs, s] = await Promise.all([
        Api.getBookings(),
        Api.getAccounts(),
        Api.getCompanySettings(),
      ]);
      setBookings(list);
      setAccounts(accs);
      setSettings(s);
      if (accs.length > 0) {
        if (!accs.some((a) => a.number === debitAccount)) {
          const bank = accs.find((a) => a.number === '1800') || accs[0];
          if (bank) setDebitAccount(bank.number);
        }
        if (!accs.some((a) => a.number === creditAccount)) {
          const rev = accs.find((a) => a.number.startsWith('4400') || a.type === 'revenue') || accs[0];
          if (rev) setCreditAccount(rev.number);
        }
      }
      updateAssignedFiscalYear(bookingDate, s);
    } finally {
      setLoading(false);
    }
  };

  const updateAssignedFiscalYear = (dateStr: string, s: CompanySettings | null) => {
    const startMonth = s?.fiscalYearStartMonth || 1;
    const d = new Date(dateStr);
    const year = d.getFullYear() || new Date().getFullYear();
    const month = (d.getMonth() + 1) || 1;

    let fy = year;
    if (startMonth > 1) {
      if (month < startMonth) {
        fy = year - 1;
      }
    }
    setAssignedFiscalYear(fy);
  };

  const handleDateChange = (newDate: string) => {
    setBookingDate(newDate);
    updateAssignedFiscalYear(newDate, settings);
  };

  // Detect if date is in transition period (e.g. within Jan-Mar, or date in previous year)
  const isTransitionDate = useMemo(() => {
    if (!bookingDate) return false;
    const d = new Date(bookingDate);
    const month = d.getMonth() + 1;
    const currentYear = new Date().getFullYear();
    // January to March or prior calendar year
    return month <= 3 || d.getFullYear() < currentYear;
  }, [bookingDate]);

  const handleCreateBookingSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const numAmount = Number(amount);
    if (!numAmount || numAmount <= 0) {
      setNewBookingError('Bitte geben Sie einen gültigen Betrag größer als 0 ein.');
      return;
    }
    if (!bookingDesc.trim()) {
      setNewBookingError('Bitte geben Sie einen Buchungstext ein.');
      return;
    }

    setIsSubmittingBooking(true);
    setNewBookingError(null);
    try {
      await Api.createBooking({
        fiscalYear: assignedFiscalYear,
        date: bookingDate,
        valueDate: bookingDate,
        description: bookingDesc.trim(),
        debitAccount: debitAccount,
        creditAccount: creditAccount,
        amount: numAmount,
        currency: 'EUR',
        receiptNumber: receiptNumber.trim(),
        taxCode: taxCode,
      });

      toast.success('Buchung erfasst', {
        description: `Buchung über ${formatCurrency(numAmount)} erfolgreich im Geschäftsjahr ${assignedFiscalYear} erfasst.`,
      });

      setIsNewBookingModalOpen(false);
      setBookingDesc('');
      setAmount('');
      setReceiptNumber('');
      await loadData();
    } catch (err: any) {
      setNewBookingError(err?.message || 'Fehler beim Erfassen der Buchung');
    } finally {
      setIsSubmittingBooking(false);
    }
  };

  const stornoMap = useMemo(() => {
    const map = new Map<number, BookingEntry>();
    for (const b of bookings) {
      if (b.stornoForId != null) {
        map.set(b.stornoForId, b);
      }
    }
    return map;
  }, [bookings]);

  const handleStorno = async () => {
    if (!stornoModalBooking || !stornoReason.trim()) return;
    setIsSubmittingStorno(true);
    setStornoError(null);
    try {
      await Api.stornoBooking(stornoModalBooking.id, stornoReason);
      toast.success('Buchung storniert', {
        description: `Gegenbuchung zu ${stornoModalBooking.bookingNumber} wurde erstellt.`,
      });
      setStornoModalBooking(null);
      setStornoReason('');
      await loadData();
    } catch (err: any) {
      setStornoError(err?.message || 'Fehler beim Stornieren der Buchung');
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
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Buchungsjournal
            <HelpTooltip
              title="Buchungsjournal"
              content="Im Journal werden alle Buchungen mit Datum, Betrag und Konten festgehalten. Korrekturen erfolgen transparent über GoBD-konforme Stornobuchungen."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Vollständige Übersicht aller erfassten Buchungssätze im gefilterten Geschäftsjahr
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-stone-100 text-stone-700 text-xs border border-stone-200">
            <Lock className="w-3.5 h-3.5 text-stone-500" />
            <span>Festschreibung aktiv</span>
          </div>

          <button
            onClick={() => setIsNewBookingModalOpen(true)}
            className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs"
          >
            <Plus className="w-3.5 h-3.5" />
            Buchung erfassen
          </button>
        </div>
      </div>

      {/* Search */}
      <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs">
        <div className="relative">
          <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Journal durchsuchen nach Beleg-Nr., Beschreibung oder Konto..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
          />
        </div>
      </div>

      {/* Bookings Table */}
      <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
              <tr>
                <th className="py-3 px-4 w-28">Nr.</th>
                <th className="py-3 px-4 w-24">Datum</th>
                <th className="py-3 px-4">Buchungstext &amp; Beleg</th>
                <th className="py-3 px-4">Soll-Konto</th>
                <th className="py-3 px-4">Haben-Konto</th>
                <th className="py-3 px-4 text-right">Betrag</th>
                <th className="py-3 px-4 text-center">Status</th>
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
                    Keine Buchungen in diesem Geschäftsjahr gefunden.
                  </td>
                </tr>
              ) : (
                filteredBookings.map((b) => {
                  const isStornoEntry = Boolean(b.isStorno || b.stornoForId != null);
                  const isStornoed = stornoMap.has(b.id);
                  const stornoChild = stornoMap.get(b.id);

                  return (
                    <tr
                      key={b.id}
                      className={`transition-colors ${
                        isStornoed
                          ? 'bg-rose-50/25 line-through text-stone-400 hover:bg-rose-50/35'
                          : isStornoEntry
                          ? 'bg-amber-50/15 text-stone-600 hover:bg-amber-50/25'
                          : 'hover:bg-stone-50/80'
                      }`}
                    >
                      <td className="py-3 px-4 font-mono font-bold text-stone-800">
                        {b.bookingNumber}
                      </td>
                      <td className="py-3 px-4 text-stone-600">
                        {formatDate(b.date)}
                      </td>
                      <td className="py-3 px-4">
                        <div className={`font-semibold ${isStornoed ? 'text-stone-500' : 'text-stone-900'}`}>
                          {b.description}
                        </div>
                        {b.receiptNumber && (
                          <div className="text-xs text-amber-800 font-mono mt-0.5 flex items-center gap-1">
                            <FileCode className="w-3 h-3 text-amber-600" />
                            <span>Beleg: {b.receiptNumber}</span>
                          </div>
                        )}
                        {isStornoed && stornoChild && (
                          <div className="text-[11px] text-rose-600 font-mono mt-0.5">
                            Storniert durch {stornoChild.bookingNumber}
                          </div>
                        )}
                      </td>
                      <td className="py-3 px-4 font-mono">
                        <div className="font-bold text-stone-900">{b.debitAccount}</div>
                        <div className="text-[11px] text-stone-500 font-sans truncate max-w-[140px]">
                          {b.debitAccountName}
                        </div>
                      </td>
                      <td className="py-3 px-4 font-mono">
                        <div className="font-bold text-stone-900">{b.creditAccount}</div>
                        <div className="text-[11px] text-stone-500 font-sans truncate max-w-[140px]">
                          {b.creditAccountName}
                        </div>
                      </td>
                      <td className="py-3 px-4 text-right font-mono font-bold text-stone-900">
                        {formatCurrency(b.amount, b.currency)}
                      </td>
                      <td className="py-3 px-4 text-center">
                        {isStornoEntry ? (
                          <div
                            className="inline-flex items-center gap-1 text-[11px] font-medium text-amber-800 bg-amber-50 px-2.5 py-0.5 rounded-full border border-amber-200/60"
                            title={b.stornoForId ? `Gegenbuchung zu ID ${b.stornoForId}` : 'Stornobuchung'}
                          >
                            <RotateCcw className="w-3.5 h-3.5 text-amber-600" />
                            <span>Storno-Korrektur</span>
                          </div>
                        ) : isStornoed ? (
                          <div
                            className="inline-flex items-center gap-1 text-[11px] font-medium text-rose-700 bg-rose-50 px-2.5 py-0.5 rounded-full border border-rose-200/60"
                            title={`Storniert durch ${stornoChild?.bookingNumber || 'Stornobuchung'}`}
                          >
                            <RotateCcw className="w-3.5 h-3.5 text-rose-500" />
                            <span>Storniert</span>
                          </div>
                        ) : (
                          <div
                            className="inline-flex items-center gap-1 text-[11px] font-medium text-emerald-700 bg-emerald-50 px-2.5 py-0.5 rounded-full border border-emerald-200/60"
                            title="Unveränderbar festgeschrieben"
                          >
                            <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
                            <span>Gesichert</span>
                          </div>
                        )}
                      </td>
                      <td className="py-3 px-4 text-right">
                        {!isStornoEntry && !isStornoed && (
                          <button
                            onClick={() => {
                              setStornoError(null);
                              setStornoReason('');
                              setStornoModalBooking(b);
                            }}
                            className="px-2.5 py-1 text-xs font-medium text-rose-700 bg-rose-50 hover:bg-rose-100 rounded-lg transition-colors inline-flex items-center gap-1 cursor-pointer"
                            title="Buchung stornieren"
                          >
                            <RotateCcw className="w-3 h-3" />
                            Stornieren
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Modal: New Booking */}
      {isNewBookingModalOpen && (
        <div className="fixed inset-0 z-50 bg-stone-900/50 backdrop-blur-xs flex items-center justify-center p-4 overflow-y-auto">
          <div className="bg-white rounded-2xl shadow-2xl max-w-lg w-full p-6 border border-stone-200 animate-in fade-in space-y-4">
            <div className="flex items-center justify-between border-b border-stone-200 pb-3">
              <h3 className="text-base font-bold text-stone-900 flex items-center gap-2">
                <Plus className="w-4 h-4 text-amber-700" />
                Buchung erfassen
              </h3>
              <button
                onClick={() => setIsNewBookingModalOpen(false)}
                className="text-stone-400 hover:text-stone-700 text-sm font-bold p-1"
              >
                ✕
              </button>
            </div>

            {newBookingError && (
              <div className="p-3 bg-rose-50 border border-rose-200 rounded-xl text-rose-700 text-xs">
                {newBookingError}
              </div>
            )}

            <form onSubmit={handleCreateBookingSubmit} className="space-y-4 text-xs">
              {/* Date & Target Fiscal Year */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Buchungsdatum:
                  </label>
                  <input
                    type="date"
                    value={bookingDate}
                    onChange={(e) => handleDateChange(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg font-mono text-xs focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20"
                    required
                  />
                </div>

                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Geschäftsjahr:
                  </label>
                  <select
                    value={assignedFiscalYear}
                    onChange={(e) => setAssignedFiscalYear(Number(e.target.value))}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg font-mono text-xs focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20"
                  >
                    <option value={assignedFiscalYear}>{assignedFiscalYear} (Automatisch)</option>
                    <option value={assignedFiscalYear - 1}>{assignedFiscalYear - 1} (Vorjahr / Abschluss)</option>
                    <option value={assignedFiscalYear + 1}>{assignedFiscalYear + 1} (Folgejahr)</option>
                  </select>
                </div>
              </div>

              {/* Boundary / Transition Period Alert */}
              {isTransitionDate && (
                <div className="p-3 bg-amber-50/80 rounded-xl border border-amber-200 text-amber-900 flex items-start gap-2">
                  <AlertTriangle className="w-4 h-4 text-amber-700 shrink-0 mt-0.5" />
                  <div className="space-y-0.5">
                    <span className="font-bold block text-[11px]">Übergangsfrist aktiv</span>
                    <p className="text-[11px] leading-snug text-stone-600">
                      Das Datum liegt im Grenzbereich / Jahreswechsel. Sie können die Buchung oben manuell dem Vorjahr (z. B. Abschlussbuchung) oder dem laufenden Geschäftsjahr zuweisen.
                    </p>
                  </div>
                </div>
              )}

              {/* Booking Text */}
              <div>
                <label className="font-semibold text-stone-700 block mb-1">
                  Buchungstext:
                </label>
                <input
                  type="text"
                  placeholder="z. B. Büromaterial, Software-Abo, Kundenzahlung..."
                  value={bookingDesc}
                  onChange={(e) => setBookingDesc(e.target.value)}
                  className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs"
                  required
                />
              </div>

              {/* Accounts */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Soll-Konto:
                  </label>
                  <select
                    value={debitAccount}
                    onChange={(e) => setDebitAccount(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono focus:outline-none focus:border-amber-600"
                  >
                    {accounts.map((acc) => (
                      <option key={acc.number} value={acc.number}>
                        {acc.number} — {acc.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Haben-Konto:
                  </label>
                  <select
                    value={creditAccount}
                    onChange={(e) => setCreditAccount(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono focus:outline-none focus:border-amber-600"
                  >
                    {accounts.map((acc) => (
                      <option key={acc.number} value={acc.number}>
                        {acc.number} — {acc.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Amount, Tax & Receipt */}
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Betrag (€):
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    placeholder="0,00"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value === '' ? '' : Number(e.target.value))}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg font-mono font-bold text-xs focus:outline-none focus:border-amber-600"
                    required
                  />
                </div>

                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Steuersatz:
                  </label>
                  <select
                    value={taxCode}
                    onChange={(e) => setTaxCode(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono focus:outline-none focus:border-amber-600"
                  >
                    <option value="NONE">Keine Steuer</option>
                    <option value="UST19">19% USt</option>
                    <option value="UST7">7% USt</option>
                    <option value="VOST19">19% VSt</option>
                    <option value="VOST7">7% VSt</option>
                  </select>
                </div>

                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Beleg-Nr. (optional):
                  </label>
                  <input
                    type="text"
                    placeholder="z. B. BE-2026-001"
                    value={receiptNumber}
                    onChange={(e) => setReceiptNumber(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg font-mono text-xs focus:outline-none focus:border-amber-600"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2 pt-3 border-t border-stone-200">
                <button
                  type="button"
                  onClick={() => setIsNewBookingModalOpen(false)}
                  className="px-4 py-2 text-stone-600 hover:bg-stone-100 rounded-lg transition-colors font-medium text-xs"
                >
                  Abbrechen
                </button>
                <button
                  type="submit"
                  disabled={isSubmittingBooking}
                  className="px-5 py-2.5 bg-amber-700 hover:bg-amber-800 text-white font-semibold rounded-lg shadow-xs transition-colors text-xs flex items-center gap-1.5"
                >
                  <CheckCircle2 className="w-4 h-4" />
                  {isSubmittingBooking ? 'Wird gebucht...' : 'Buchung speichern'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Storno Modal Dialog */}
      {stornoModalBooking && (
        <div className="fixed inset-0 z-50 bg-stone-900/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full p-6 border border-stone-200 animate-in fade-in zoom-in-95 duration-150">
            <h3 className="text-base font-bold text-stone-900 flex items-center gap-2">
              <RotateCcw className="w-4 h-4 text-rose-600" />
              Buchung stornieren
            </h3>
            <p className="text-xs text-stone-500 mt-1">
              Bereits erfasste Buchungen können nicht gelöscht werden. Es wird eine
              transparente Gegenbuchung erzeugt, die den Betrag neutralisiert.
            </p>

            <div className="my-4 p-3.5 bg-stone-50 rounded-xl text-xs space-y-1">
              <div className="text-stone-600 font-mono">
                Buchung: <span className="font-bold text-stone-900">{stornoModalBooking.bookingNumber}</span>
              </div>
              <div className="text-stone-600">
                Text: <span className="text-stone-900 font-medium">{stornoModalBooking.description}</span>
              </div>
              <div className="text-stone-600 font-mono">
                Betrag: <span className="font-bold text-stone-900">{formatCurrency(stornoModalBooking.amount)}</span>
              </div>
            </div>

            {stornoError && (
              <div className="p-3 bg-rose-50 border border-rose-200 rounded-xl text-rose-700 text-xs">
                {stornoError}
              </div>
            )}

            <div className="space-y-2">
              <label className="text-xs font-medium text-stone-700 block">
                Grund für die Stornierung:
              </label>
              <textarea
                rows={3}
                placeholder="z. B. Falsches Konto gewählt, Rechnungskorrektur..."
                value={stornoReason}
                onChange={(e) => setStornoReason(e.target.value)}
                className="w-full text-xs p-2.5 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-rose-500/20 focus:border-rose-500"
              />
            </div>

            <div className="flex justify-end gap-2 mt-6">
              <button
                type="button"
                onClick={() => setStornoModalBooking(null)}
                className="px-3.5 py-2 text-xs font-medium text-stone-600 hover:bg-stone-100 rounded-lg transition-colors"
              >
                Abbrechen
              </button>
              <button
                type="button"
                disabled={!stornoReason.trim() || isSubmittingStorno}
                onClick={handleStorno}
                className="px-4 py-2 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-700 disabled:opacity-50 rounded-lg shadow-xs transition-colors"
              >
                {isSubmittingStorno ? 'Wird gebucht...' : 'Buchung jetzt stornieren'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
