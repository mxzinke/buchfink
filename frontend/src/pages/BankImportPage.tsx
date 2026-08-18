import React, { useEffect, useState } from 'react';
import { toast } from 'sonner';
import {
  Landmark,
  Upload,
  CheckCircle2,
  Sparkles,
  Check,
} from 'lucide-react';
import { BankTransaction, Account } from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const BankImportPage: React.FC = () => {
  const [transactions, setTransactions] = useState<BankTransaction[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [activeTx, setActiveTx] = useState<BankTransaction | null>(null);

  // Match form
  const [selectedDebit, setSelectedDebit] = useState('1800');
  const [selectedCredit, setSelectedCredit] = useState('4400');
  const [receiptNumber, setReceiptNumber] = useState('');
  const [bookingDesc, setBookingDesc] = useState('');
  const [isBooking, setIsBooking] = useState(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [txs, accs] = await Promise.all([
        Api.getBankTransactions(),
        Api.getAccounts(),
      ]);
      setTransactions(txs);
      setAccounts(accs);
    } catch (e) {
      console.error(e);
    }
  };

  const handleSelectTx = (tx: BankTransaction) => {
    setActiveTx(tx);
    setReceiptNumber('');
    setBookingDesc(`${tx.remittanceInfo || 'Bankzahlung'} (${tx.counterpartyName})`);

    if (tx.amount > 0) {
      // Einnahme: Soll Bank 1800 an Haben Erlöse (z.B. 4400)
      setSelectedDebit('1800');
      setSelectedCredit(tx.suggestedAccount || '4400');
    } else {
      // Ausgabe: Soll Aufwand (z.B. 6800) an Haben Bank 1800
      setSelectedDebit(tx.suggestedAccount || '6800');
      setSelectedCredit('1800');
    }
  };

  const handleConfirmBooking = async () => {
    if (!activeTx) return;
    setIsBooking(true);
    try {
      await Api.matchAndBookTransaction(
        activeTx.id,
        selectedDebit,
        selectedCredit,
        receiptNumber,
        bookingDesc
      );
      setSuccessMessage(`Zahlung erfolgreich verbucht (${selectedDebit} an ${selectedCredit}).`);
      toast.success('Zahlung verbucht', {
        description: `Erfolgreich verbucht (${selectedDebit} an ${selectedCredit}).`,
      });
      setActiveTx(null);
      await loadData();
      setTimeout(() => setSuccessMessage(null), 4000);
    } finally {
      setIsBooking(false);
    }
  };

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Bank & Zahlungsabgleich
            <HelpTooltip
              title="Automatisierter Zahlungsabgleich"
              content="Buchfink schlägt Ihnen für jede Bankzahlung automatisch das passende Gegenkonto vor. Ein Klick genügt, um die Zahlung im Journal zu erfassen."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Kontoauszüge einlesen und Zahlungen mit einem Klick verbuchen
          </p>
        </div>

        <div className="flex gap-2">
          <label className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors cursor-pointer shadow-xs">
            <Upload className="w-3.5 h-3.5" />
            Kontoauszug importieren (CAMT.053)
            <input
              type="file"
              accept=".xml"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) {
                  const reader = new FileReader();
                  reader.onload = async () => {
                    if (typeof reader.result === 'string') {
                      await Api.importCAMT053XML(reader.result);
                      await loadData();
                    }
                  };
                  reader.readAsText(file);
                }
              }}
            />
          </label>
        </div>
      </div>

      {/* Success alert */}
      {successMessage && (
        <div className="p-4 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-800 flex items-center gap-2 animate-in fade-in">
          <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
          <span>{successMessage}</span>
        </div>
      )}

      {/* 2-Column Layout: Transactions List & Matching Panel */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left: Transaction Table */}
        <div className="lg:col-span-7 bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
          <div className="p-4 border-b border-stone-200 bg-stone-50/60 flex items-center justify-between">
            <h3 className="text-xs font-bold text-stone-900 uppercase tracking-wider">
              Bankumsätze ({transactions.length})
            </h3>
            <span className="text-xs text-stone-500">Geschäftskonto</span>
          </div>

          <div className="divide-y divide-stone-100 max-h-[600px] overflow-y-auto">
            {transactions.length === 0 ? (
              <div className="p-12 text-center text-stone-400 text-xs">
                Keine Bankumsätze vorhanden. Bitte importieren Sie einen Kontoauszug im CAMT.053 XML-Format.
              </div>
            ) : (
              transactions.map((tx) => {
                const isSelected = activeTx?.id === tx.id;
                const isMatched = tx.matchStatus === 'matched';

                return (
                  <div
                    key={tx.id}
                    onClick={() => !isMatched && handleSelectTx(tx)}
                    className={`p-4 transition-all cursor-pointer flex items-center justify-between gap-4 ${
                      isSelected
                        ? 'bg-amber-50/80 border-l-4 border-amber-700'
                        : isMatched
                        ? 'bg-stone-50/50 opacity-60'
                        : 'hover:bg-stone-50'
                    }`}
                  >
                    <div className="space-y-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-bold text-stone-900 truncate">
                          {tx.counterpartyName || 'Unbekannter Absender/Empfänger'}
                        </span>
                        {isMatched ? (
                          <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-100 text-emerald-800 flex items-center gap-1">
                            <Check className="w-2.5 h-2.5" /> Verbucht
                          </span>
                        ) : tx.suggestedAccount ? (
                          <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-amber-100 text-amber-800 flex items-center gap-1">
                            <Sparkles className="w-2.5 h-2.5" /> Vorschlag: {tx.suggestedAccount}
                          </span>
                        ) : null}
                      </div>

                      <div className="text-xs text-stone-500 truncate">
                        {tx.remittanceInfo}
                      </div>

                      <div className="text-[11px] text-stone-400">
                        {formatDate(tx.valueDate)} &bull; {tx.counterpartyIban}
                      </div>
                    </div>

                    <div className="text-right shrink-0">
                      <div
                        className={`text-xs font-mono font-bold ${
                          tx.amount > 0 ? 'text-emerald-700' : 'text-stone-900'
                        }`}
                      >
                        {tx.amount > 0 ? '+' : ''}
                        {formatCurrency(tx.amount, tx.currency)}
                      </div>
                      {!isMatched && (
                        <span className="text-xs text-amber-700 font-medium hover:underline">
                          Zuordnen &rarr;
                        </span>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Right: Booking Matcher Assistant */}
        <div className="lg:col-span-5 bg-white rounded-xl border border-stone-200/80 shadow-xs p-6 space-y-5 sticky top-20">
          <div className="border-b border-stone-200 pb-3">
            <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-amber-700" />
              Zahlung zuordnen
            </h3>
            <p className="text-xs text-stone-500 mt-0.5">
              Wählen Sie die passenden Konten für die Verbuchung aus.
            </p>
          </div>

          {!activeTx ? (
            <div className="py-12 text-center text-stone-400 text-xs">
              <Landmark className="w-8 h-8 text-stone-300 mx-auto mb-2" />
              Wählen Sie links eine offene Zahlung aus, um sie mit einem Klick zu verbuchen.
            </div>
          ) : (
            <div className="space-y-4 text-xs">
              <div className="p-3 bg-amber-50/60 rounded-lg border border-amber-200/60 space-y-1">
                <div className="flex justify-between font-medium text-amber-900">
                  <span>Betrag:</span>
                  <span className="font-bold font-mono">
                    {formatCurrency(Math.abs(activeTx.amount), activeTx.currency)}
                  </span>
                </div>
                <div className="text-stone-600">
                  <span className="font-medium text-stone-700">Partner:</span> {activeTx.counterpartyName}
                </div>
                <div className="text-stone-500 text-xs truncate">
                  <span className="font-medium text-stone-700">Verwendungszweck:</span> {activeTx.remittanceInfo}
                </div>
              </div>

              {/* Debit Account */}
              <div>
                <label className="font-semibold text-stone-700 block mb-1">
                  Soll-Konto:
                </label>
                <select
                  value={selectedDebit}
                  onChange={(e) => setSelectedDebit(e.target.value)}
                  className="w-full p-2 text-xs bg-stone-50 border border-stone-200 rounded-lg text-stone-800 focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20"
                >
                  {accounts.map((acc) => (
                    <option key={acc.number} value={acc.number}>
                      {acc.number} — {acc.name} ({acc.category})
                    </option>
                  ))}
                </select>
              </div>

              {/* Credit Account */}
              <div>
                <label className="font-semibold text-stone-700 block mb-1">
                  Haben-Konto:
                </label>
                <select
                  value={selectedCredit}
                  onChange={(e) => setSelectedCredit(e.target.value)}
                  className="w-full p-2 text-xs bg-stone-50 border border-stone-200 rounded-lg text-stone-800 focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20"
                >
                  {accounts.map((acc) => (
                    <option key={acc.number} value={acc.number}>
                      {acc.number} — {acc.name} ({acc.category})
                    </option>
                  ))}
                </select>
              </div>

              {/* Receipt Number */}
              <div>
                <label className="font-semibold text-stone-700 block mb-1">
                  Belegnummer / Rechnungs-Nr. (optional):
                </label>
                <input
                  type="text"
                  placeholder="z. B. RE-2026-001"
                  value={receiptNumber}
                  onChange={(e) => setReceiptNumber(e.target.value)}
                  className="w-full p-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
                />
              </div>

              {/* Booking Text */}
              <div>
                <label className="font-semibold text-stone-700 block mb-1">
                  Buchungstext:
                </label>
                <input
                  type="text"
                  value={bookingDesc}
                  onChange={(e) => setBookingDesc(e.target.value)}
                  className="w-full p-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
                />
              </div>

              <div className="pt-2">
                <button
                  type="button"
                  disabled={isBooking}
                  onClick={handleConfirmBooking}
                  className="w-full py-2.5 rounded-lg bg-amber-700 text-white font-semibold hover:bg-amber-800 transition-colors shadow-xs flex items-center justify-center gap-2"
                >
                  <CheckCircle2 className="w-4 h-4" />
                  {isBooking ? 'Wird verbucht...' : 'Zahlung jetzt verbuchen'}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
