import React, { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AlertCircle, ArrowDownLeft, ArrowUpRight, Ban, Landmark, RefreshCw, Upload } from 'lucide-react';
import type {
  Account,
  AllocationRequest,
  BankTransaction,
  DifferenceKind,
  DifferenceKindInfo,
  OpenItem,
} from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, parseCents } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';
import { inputClass } from '../components/Form';

/**
 * Bank & Zahlungen.
 *
 * Ein Kontoauszug kennt zwei Fälle. Entweder gehört die Zahlung zu einem offenen
 * Posten — dann wird sie zugeordnet, samt Skonto oder Gebühr. Oder es gibt keinen
 * Beleg, etwa bei Kontoführungsentgelten — dann wird direkt gegen ein Konto
 * gebucht. Beides steht hier nebeneinander, statt beides über dieselbe Maske zu
 * zwingen.
 */

export const BankImportPage: React.FC = () => {
  const [transactions, setTransactions] = useState<BankTransaction[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [openItems, setOpenItems] = useState<OpenItem[]>([]);
  const [differenceKinds, setDifferenceKinds] = useState<DifferenceKindInfo[]>([]);
  const [importAccount, setImportAccount] = useState('1800');
  const [active, setActive] = useState<BankTransaction | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [txs, accs, payAccs, items, kinds] = await Promise.all([
        Api.getBankTransactions(),
        Api.getAccounts(),
        Api.getPaymentAccounts(),
        Api.getOpenItems(),
        Api.getDifferenceKinds(),
      ]);
      setTransactions(txs);
      setAccounts(accs);
      setPaymentAccounts(payAccs);
      setOpenItems(items);
      setDifferenceKinds(kinds);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function importFile(file: File) {
    setError(null);
    try {
      const content = await file.text();
      const count = await Api.importCAMT(content, importAccount);
      toast.success(`${count} Umsätze importiert`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  const unmatched = transactions.filter((t) => t.matchStatus === 'unmatched');

  return (
    <div className="p-4 sm:p-6 space-y-5 max-w-6xl mx-auto">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold text-stone-900 tracking-tight">Bank & Zahlungen</h1>
          <p className="text-sm text-stone-600">
            Kontoauszüge einlesen und Zahlungen den offenen Posten zuordnen.
          </p>
        </div>

        <div className="flex items-end gap-2">
          <label className="block">
            <span className="block text-xs font-medium text-stone-600 mb-1">Bankkonto des Auszugs</span>
            <select
              value={importAccount}
              onChange={(e) => setImportAccount(e.target.value)}
              className={`${inputClass} w-56`}
            >
              {paymentAccounts.map((a) => (
                <option key={a.number} value={a.number}>
                  {a.number} · {a.name}
                </option>
              ))}
            </select>
          </label>
          <label className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-amber-700 text-white text-sm font-medium hover:bg-amber-800 cursor-pointer">
            <Upload className="w-4 h-4" />
            CAMT.053 importieren
            <input
              type="file"
              accept=".xml"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) void importFile(file);
                e.target.value = '';
              }}
            />
          </label>
        </div>
      </header>

      {error && (
        <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-xl p-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <p className="text-xs text-stone-500 bg-stone-50 border border-stone-200 rounded-lg p-3 leading-relaxed">
        Buchfink schlägt bewusst kein Gegenkonto vor. Aus dem Verwendungszweck ein Aufwandskonto zu raten
        wäre eine unprüfbare Vermutung an der Stelle, an der die Kontierung entschieden wird — und für
        diese Entscheidung haftet das Unternehmen.
      </p>

      {loading ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-stone-400">
          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
          Bankumsätze werden geladen…
        </div>
      ) : unmatched.length === 0 ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-sm text-stone-500">
          <Landmark className="w-6 h-6 mx-auto mb-2 text-stone-300" />
          Keine offenen Bankumsätze. Alle importierten Zahlungen sind zugeordnet.
        </div>
      ) : (
        <div className="bg-white rounded-2xl border border-stone-200/80 shadow-xs divide-y divide-stone-100">
          {unmatched.map((tx) => (
            <button
              key={tx.id}
              onClick={() => setActive(tx)}
              className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-amber-50/40"
            >
              {tx.amount > 0 ? (
                <ArrowDownLeft className="w-4 h-4 text-emerald-600 shrink-0" />
              ) : (
                <ArrowUpRight className="w-4 h-4 text-rose-600 shrink-0" />
              )}
              <div className="min-w-0 flex-1">
                <div className="text-sm text-stone-900 truncate">{tx.counterpartyName || '—'}</div>
                <div className="text-xs text-stone-500 truncate">{tx.remittanceInfo}</div>
              </div>
              <div className="text-xs text-stone-500 shrink-0">{formatDate(tx.bookingDate)}</div>
              <div
                className={`font-mono text-sm tabular-nums shrink-0 ${
                  tx.amount > 0 ? 'text-emerald-700' : 'text-stone-900'
                }`}
              >
                {formatCents(tx.amount, tx.currency)}
              </div>
            </button>
          ))}
        </div>
      )}

      {active && (
        <AssignDialog
          tx={active}
          accounts={accounts}
          openItems={openItems}
          differenceKinds={differenceKinds}
          onClose={() => setActive(null)}
          onDone={async () => {
            setActive(null);
            await load();
          }}
        />
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

const AssignDialog: React.FC<{
  tx: BankTransaction;
  accounts: Account[];
  openItems: OpenItem[];
  differenceKinds: DifferenceKindInfo[];
  onClose: () => void;
  onDone: () => void;
}> = ({ tx, accounts, openItems, differenceKinds, onClose, onDone }) => {
  const [mode, setMode] = useState<'open_item' | 'direct'>('open_item');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Zahlungseingänge gleichen Forderungen aus, Ausgänge Verbindlichkeiten.
  const relevant = useMemo(
    () => openItems.filter((i) => (tx.amount > 0 ? i.contactType === 'customer' : i.contactType === 'vendor')),
    [openItems, tx.amount]
  );

  const [selected, setSelected] = useState<Record<number, { amount: string; kind: DifferenceKind; diff: string }>>(
    {}
  );
  const [counterAccount, setCounterAccount] = useState('');
  const [description, setDescription] = useState(`${tx.counterpartyName} – ${tx.remittanceInfo}`);

  const postable = useMemo(
    () => accounts.filter((a) => !a.isRange && !a.isReserved && a.kontenklasse !== 8),
    [accounts]
  );

  const allocations: AllocationRequest[] = Object.entries(selected).map(([entryId, value]) => ({
    openItemEntryId: Number(entryId),
    settledAmount: parseCents(value.amount) ?? 0,
    differenceKind: value.kind,
    differenceAmount: value.kind === 'none' ? 0 : parseCents(value.diff) ?? 0,
  }));

  const cashTotal = allocations.reduce((sum, a) => {
    if (a.differenceKind === 'bank_fee') return sum + a.settledAmount + a.differenceAmount;
    if (a.differenceKind === 'none') return sum + a.settledAmount;
    return sum + a.settledAmount - a.differenceAmount;
  }, 0);
  const statementAmount = Math.abs(tx.amount);
  const matches = cashTotal === statementAmount && allocations.length > 0;

  function toggle(item: OpenItem) {
    setSelected((prev) => {
      if (prev[item.entryId]) {
        const next = { ...prev };
        delete next[item.entryId];
        return next;
      }
      return {
        ...prev,
        [item.entryId]: { amount: formatCents(item.openAmount, ''), kind: 'none', diff: '' },
      };
    });
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      if (mode === 'open_item') {
        await Api.settlePayment({
          bankTxId: tx.id,
          paymentAccount: tx.ledgerAccount,
          paymentDate: tx.bookingDate,
          valueDate: tx.valueDate,
          allocations,
        });
        toast.success('Zahlung zugeordnet');
      } else {
        await Api.bookBankTransactionDirect(tx.id, counterAccount.trim(), description);
        toast.success('Bankumsatz gebucht');
      }
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function ignore() {
    setBusy(true);
    setError(null);
    try {
      await Api.ignoreBankTransaction(tx.id);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-stone-900/40 flex items-start justify-center p-4 overflow-y-auto z-50">
      <div className="bg-white rounded-2xl border border-stone-200 shadow-xl w-full max-w-3xl my-8">
        <div className="px-5 py-3 border-b border-stone-100">
          <h2 className="font-semibold text-stone-900">Bankumsatz zuordnen</h2>
          <p className="text-xs text-stone-500 mt-0.5">
            {formatDate(tx.bookingDate)} · {tx.counterpartyName} ·{' '}
            <span className="font-mono">{formatCents(tx.amount, tx.currency)}</span> auf Konto{' '}
            {tx.ledgerAccount}
          </p>
        </div>

        <div className="px-5 pt-4">
          <div className="flex gap-1 border-b border-stone-200">
            {(
              [
                { id: 'open_item', label: 'Offenen Posten ausgleichen' },
                { id: 'direct', label: 'Ohne Beleg direkt buchen' },
              ] as const
            ).map(({ id, label }) => (
              <button
                key={id}
                onClick={() => setMode(id)}
                className={`px-3 py-2 text-sm font-medium border-b-2 -mb-px ${
                  mode === id
                    ? 'border-amber-600 text-amber-800'
                    : 'border-transparent text-stone-500 hover:text-stone-800'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="p-5 space-y-4">
          {mode === 'open_item' ? (
            <>
              {relevant.length === 0 ? (
                <p className="text-sm text-stone-500">
                  Es gibt keine passenden offenen Posten. Wenn zu dieser Zahlung kein Beleg gehört, buche
                  sie über den zweiten Reiter direkt.
                </p>
              ) : (
                <div className="space-y-2">
                  {relevant.map((item) => {
                    const entry = selected[item.entryId];
                    return (
                      <div
                        key={item.entryId}
                        className={`rounded-lg border p-3 ${
                          entry ? 'border-amber-300 bg-amber-50/40' : 'border-stone-200'
                        }`}
                      >
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={Boolean(entry)}
                            onChange={() => toggle(item)}
                            className="accent-amber-700"
                          />
                          <span className="text-sm text-stone-900 flex-1">
                            {item.documentNumber || item.entryNumber} · {item.contactName}
                          </span>
                          <span className="text-xs text-stone-500">
                            fällig {formatDate(item.dueDate)}
                          </span>
                          <span className="font-mono text-sm tabular-nums">
                            {formatCents(item.openAmount)}
                          </span>
                        </label>

                        {entry && (
                          <div className="mt-3 grid grid-cols-1 sm:grid-cols-3 gap-2">
                            <label className="block">
                              <span className="block text-[11px] text-stone-500 mb-1">Ausgleichsbetrag</span>
                              <input
                                value={entry.amount}
                                onChange={(e) =>
                                  setSelected((prev) => ({
                                    ...prev,
                                    [item.entryId]: { ...prev[item.entryId], amount: e.target.value },
                                  }))
                                }
                                className={`${inputClass} text-right font-mono`}
                              />
                            </label>
                            <label className="block">
                              <span className="block text-[11px] text-stone-500 mb-1">Differenz</span>
                              <select
                                value={entry.kind}
                                onChange={(e) =>
                                  setSelected((prev) => ({
                                    ...prev,
                                    [item.entryId]: {
                                      ...prev[item.entryId],
                                      kind: e.target.value as DifferenceKind,
                                    },
                                  }))
                                }
                                className={inputClass}
                              >
                                {differenceKinds.map((k) => (
                                  <option key={k.kind} value={k.kind} title={k.hint}>
                                    {k.label}
                                  </option>
                                ))}
                              </select>
                            </label>
                            {entry.kind !== 'none' && (
                              <label className="block">
                                <span className="block text-[11px] text-stone-500 mb-1">
                                  Betrag der Differenz
                                </span>
                                <input
                                  value={entry.diff}
                                  onChange={(e) =>
                                    setSelected((prev) => ({
                                      ...prev,
                                      [item.entryId]: { ...prev[item.entryId], diff: e.target.value },
                                    }))
                                  }
                                  placeholder="0,00"
                                  className={`${inputClass} text-right font-mono`}
                                />
                              </label>
                            )}
                            {entry.kind === 'skonto' && (
                              <p className="sm:col-span-3 text-[11px] text-stone-500">
                                Skonto brutto eingeben. Buchfink teilt den Betrag in Entgelt und Steuer und
                                korrigiert die Umsatz- bzw. Vorsteuer nach § 17 UStG mit
                                {' '}
                                {item.taxRate ? `${item.taxRate / 100} %` : 'dem Satz des Belegs'}.
                              </p>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}

                  <div
                    className={`flex justify-between items-center text-sm rounded-lg px-3 py-2 border ${
                      matches
                        ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
                        : 'bg-stone-50 border-stone-200 text-stone-600'
                    }`}
                  >
                    <span className="font-medium">
                      {matches
                        ? 'Zuordnung passt zum Kontoauszug'
                        : `Noch ${formatCents(statementAmount - cashTotal)} offen`}
                    </span>
                    <span className="font-mono tabular-nums">
                      {formatCents(cashTotal)} von {formatCents(statementAmount)}
                    </span>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="space-y-3">
              <p className="text-xs text-stone-500 bg-stone-50 border border-stone-200 rounded-lg p-3">
                Für Zinsen, Kontoführungsentgelte, Privatentnahmen oder Umbuchungen zwischen eigenen
                Konten. Die Bankseite kommt aus dem Kontoauszug — die Richtung kann nicht vertippt werden.
              </p>
              <label className="block">
                <span className="block text-xs font-medium text-stone-600 mb-1">
                  Gegenkonto
                  <HelpTooltip
                    title="Gegenkonto"
                    content="Das Konto, gegen das der Bankumsatz gebucht wird. Buchfink prüft, ob es im SKR04 existiert und bebucht werden darf."
                  />
                </span>
                <input
                  list="bank-postable-accounts"
                  value={counterAccount}
                  onChange={(e) => setCounterAccount(e.target.value)}
                  placeholder="z. B. 6855"
                  className={`${inputClass} font-mono`}
                />
                <datalist id="bank-postable-accounts">
                  {postable.map((a) => (
                    <option key={a.number} value={a.number}>
                      {a.name}
                    </option>
                  ))}
                </datalist>
                <span className="block text-xs text-stone-500 mt-1">
                  {postable.find((a) => a.number === counterAccount)?.name ?? ''}
                </span>
              </label>
              <label className="block">
                <span className="block text-xs font-medium text-stone-600 mb-1">Buchungstext</span>
                <input
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className={inputClass}
                />
              </label>
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-lg p-3">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 px-5 py-3 border-t border-stone-100">
          <button
            onClick={ignore}
            disabled={busy}
            className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg text-stone-500 hover:text-stone-800"
          >
            <Ban className="w-4 h-4" />
            Nicht buchen
          </button>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="px-3 py-2 text-sm rounded-lg border border-stone-200 text-stone-700 hover:bg-stone-50"
            >
              Abbrechen
            </button>
            <button
              onClick={submit}
              disabled={busy || (mode === 'open_item' ? !matches : !counterAccount.trim())}
              className="px-3 py-2 text-sm rounded-lg bg-amber-700 text-white hover:bg-amber-800 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {busy ? 'Wird gebucht…' : 'Buchen'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
