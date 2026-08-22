// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

import React, { useEffect, useMemo, useState } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Plus,
  RefreshCw,
  Search,
  ShieldCheck,
  Trash2,
  Undo2,
  X,
} from 'lucide-react';
import type { Account, IntegrityCheckResult, JournalEntry, JournalLine, Side } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, formatDateRange, formatShortHash, parseCents } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

const SOURCE_LABELS: Record<string, string> = {
  manual: 'Manuell',
  receipt: 'Eingangsbeleg',
  invoice: 'Ausgangsrechnung',
  payment: 'Zahlung',
  opening: 'Eröffnungsbilanz',
  depreciation: 'Abschreibung',
  closing: 'Abschlussbuchung',
};

interface DraftLine {
  side: Side;
  account: string;
  amount: string;
}

const emptyDraft = (): DraftLine[] => [
  { side: 'S', account: '', amount: '' },
  { side: 'H', account: '', amount: '' },
];

export const JournalPage: React.FC = () => {
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);

  const [showForm, setShowForm] = useState(false);
  const [reversing, setReversing] = useState<JournalEntry | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [entryList, accountList] = await Promise.all([Api.getJournalEntries(), Api.getAccounts()]);
      setEntries(entryList);
      setAccounts(accountList);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function runIntegrityCheck() {
    setError(null);
    try {
      setIntegrity(await Api.verifyIntegrity());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  const accountNames = useMemo(() => {
    const map = new Map<string, string>();
    for (const account of accounts) map.set(account.number, account.name);
    return map;
  }, [accounts]);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return entries;
    return entries.filter(
      (entry) =>
        entry.entryNumber.toLowerCase().includes(query) ||
        entry.description.toLowerCase().includes(query) ||
        (entry.documentNumber ?? '').toLowerCase().includes(query) ||
        entry.lines.some((line) => line.account.includes(query))
    );
  }, [entries, search]);

  return (
    <div className="p-4 sm:p-6 space-y-5 max-w-6xl mx-auto">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold text-stone-900 tracking-tight">Journal</h1>
          <p className="text-sm text-stone-600">
            Alle Buchungen des Geschäftsjahres in der Reihenfolge ihrer Erfassung. Jede Buchung ist über
            eine Hash-Kette mit ihrer Vorgängerin verbunden und nicht mehr veränderbar.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={runIntegrityCheck}
            className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg border border-stone-200 text-stone-700 hover:bg-stone-50"
          >
            <ShieldCheck className="w-4 h-4" />
            Integrität prüfen
          </button>
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg bg-amber-700 text-white hover:bg-amber-800"
          >
            <Plus className="w-4 h-4" />
            Neue Buchung
          </button>
        </div>
      </header>

      {error && (
        <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-xl p-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {integrity && (
        <div
          className={`flex items-start gap-2 rounded-xl p-3 text-sm border ${
            integrity.isValid
              ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
              : 'bg-rose-50 border-rose-200 text-rose-800'
          }`}
        >
          {integrity.isValid ? (
            <CheckCircle2 className="w-4 h-4 mt-0.5 shrink-0" />
          ) : (
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
          )}
          <div className="space-y-0.5">
            <div>{integrity.message}</div>
            <div className="text-xs opacity-80">
              Geprüft am {integrity.checkedAt} · Kettenkopf{' '}
              <span className="font-mono">{formatShortHash(integrity.lastVerifiedHash)}</span>
            </div>
          </div>
          <button onClick={() => setIntegrity(null)} className="ml-auto shrink-0 opacity-60 hover:opacity-100">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      <div className="relative">
        <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-stone-400" />
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Buchungsnummer, Text, Belegnummer oder Konto"
          className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-stone-200 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none"
        />
      </div>

      {loading ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-stone-400">
          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
          Buchungen werden geladen…
        </div>
      ) : filtered.length === 0 ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-sm text-stone-500">
          {entries.length === 0
            ? 'Noch keine Buchungen im aktiven Geschäftsjahr.'
            : 'Keine Buchung passt zur Suche.'}
        </div>
      ) : (
        <div className="bg-white rounded-2xl border border-stone-200/80 shadow-xs divide-y divide-stone-100">
          {filtered.map((entry) => (
            <EntryRow
              key={entry.id}
              entry={entry}
              accountNames={accountNames}
              expanded={expanded[entry.id] ?? false}
              onToggle={() => setExpanded((prev) => ({ ...prev, [entry.id]: !prev[entry.id] }))}
              onReverse={() => setReversing(entry)}
            />
          ))}
        </div>
      )}

      {showForm && (
        <BookingForm
          accounts={accounts}
          onClose={() => setShowForm(false)}
          onSaved={async () => {
            setShowForm(false);
            await load();
          }}
        />
      )}

      {reversing && (
        <ReverseDialog
          entry={reversing}
          onClose={() => setReversing(null)}
          onDone={async () => {
            setReversing(null);
            await load();
          }}
        />
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

const EntryRow: React.FC<{
  entry: JournalEntry;
  accountNames: Map<string, string>;
  expanded: boolean;
  onToggle: () => void;
  onReverse: () => void;
}> = ({ entry, accountNames, expanded, onToggle, onReverse }) => {
  const gross = entry.lines
    .filter((l) => l.side === 'S')
    .reduce((sum, l) => sum + l.amount, 0);
  const isReversal = entry.kind === 'reversal';

  return (
    <div className={isReversal ? 'bg-rose-50/40' : undefined}>
      <div className="flex items-center gap-3 px-4 py-3">
        <button onClick={onToggle} className="text-stone-400 hover:text-stone-700 shrink-0">
          {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
        </button>

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs text-stone-600">{entry.entryNumber}</span>
            <span className="text-sm text-stone-900 truncate">{entry.description}</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded border border-stone-200 text-stone-500">
              {SOURCE_LABELS[entry.source] ?? entry.source}
            </span>
            {isReversal && (
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-rose-100 text-rose-700 font-medium">
                Generalumkehr
              </span>
            )}
          </div>
          <div className="text-xs text-stone-500 mt-0.5">
            Buchung {formatDate(entry.bookingDate)} · Beleg {formatDate(entry.documentDate)} · Leistung{' '}
            {formatDateRange(entry.serviceDateFrom, entry.serviceDateTo)}
            {entry.documentNumber ? ` · ${entry.documentNumber}` : ''}
          </div>
        </div>

        <div className="text-right shrink-0">
          <div className="font-mono text-sm tabular-nums text-stone-900">{formatCents(gross)}</div>
          <div className="text-[10px] text-stone-400">{entry.lines.length} Zeilen</div>
        </div>

        {!isReversal && (
          <button
            onClick={onReverse}
            title="Buchung per Generalumkehr stornieren"
            className="shrink-0 text-stone-400 hover:text-rose-600 p-1"
          >
            <Undo2 className="w-4 h-4" />
          </button>
        )}
      </div>

      {expanded && (
        <div className="px-4 pb-3">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-[10px] uppercase tracking-wide text-stone-400">
                <th className="text-left font-medium py-1">Seite</th>
                <th className="text-left font-medium py-1">Konto</th>
                <th className="text-left font-medium py-1">Bezeichnung</th>
                <th className="text-right font-medium py-1">Betrag</th>
              </tr>
            </thead>
            <tbody>
              {entry.lines.map((line) => (
                <LineRow key={line.id || `${line.position}`} line={line} accountNames={accountNames} />
              ))}
            </tbody>
          </table>
          <div className="mt-2 pt-2 border-t border-stone-100 text-[10px] text-stone-400 font-mono">
            Hash {formatShortHash(entry.entryHash)} · Vorgänger {formatShortHash(entry.previousHash)}
            {entry.postingRuleVersion ? ` · Kontierungsregeln ${entry.postingRuleVersion}` : ''}
          </div>
        </div>
      )}
    </div>
  );
};

const LineRow: React.FC<{ line: JournalLine; accountNames: Map<string, string> }> = ({
  line,
  accountNames,
}) => (
  <tr className="border-t border-stone-50">
    <td className="py-1.5">
      <span
        className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${
          line.side === 'S' ? 'bg-sky-50 text-sky-700' : 'bg-emerald-50 text-emerald-700'
        }`}
      >
        {line.side === 'S' ? 'Soll' : 'Haben'}
      </span>
    </td>
    <td className="py-1.5 font-mono text-stone-700">{line.account}</td>
    <td className="py-1.5 text-stone-600">
      {line.accountName || accountNames.get(line.account) || '—'}
      {line.taxKey && (
        <span className="ml-1.5 text-[10px] text-stone-400 font-mono">{line.taxKey}</span>
      )}
    </td>
    <td className="py-1.5 text-right font-mono tabular-nums text-stone-900">
      {formatCents(line.amount)}
    </td>
  </tr>
);

// -------------------------------------------------------------------------

/**
 * Erfassung eines Buchungssatzes mit beliebig vielen Zeilen.
 *
 * Die Summenzeile zeigt Soll und Haben laufend an. Gespeichert werden kann erst,
 * wenn beide übereinstimmen — dieselbe Regel, die das Backend beim Buchen
 * erzwingt, hier nur früher sichtbar.
 */
const BookingForm: React.FC<{
  accounts: Account[];
  onClose: () => void;
  onSaved: () => void;
}> = ({ accounts, onClose, onSaved }) => {
  const today = new Date().toISOString().split('T')[0];
  const [bookingDate, setBookingDate] = useState(today);
  const [documentDate, setDocumentDate] = useState(today);
  const [serviceFrom, setServiceFrom] = useState(today);
  const [serviceTo, setServiceTo] = useState(today);
  const [description, setDescription] = useState('');
  const [documentNumber, setDocumentNumber] = useState('');
  const [lines, setLines] = useState<DraftLine[]>(emptyDraft);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const parsed = lines.map((line) => ({ ...line, cents: parseCents(line.amount) ?? 0 }));
  const debitTotal = parsed.filter((l) => l.side === 'S').reduce((s, l) => s + l.cents, 0);
  const creditTotal = parsed.filter((l) => l.side === 'H').reduce((s, l) => s + l.cents, 0);
  const balanced = debitTotal === creditTotal && debitTotal > 0;

  const postable = useMemo(
    () => accounts.filter((a) => !a.isRange && !a.isReserved && a.kontenklasse !== 8),
    [accounts]
  );

  function updateLine(index: number, patch: Partial<DraftLine>) {
    setLines((prev) => prev.map((line, i) => (i === index ? { ...line, ...patch } : line)));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSaving(true);
    try {
      await Api.postJournalEntry({
        bookingDate,
        documentDate,
        serviceDateFrom: serviceFrom,
        serviceDateTo: serviceTo,
        description,
        documentNumber,
        source: 'manual',
        lines: parsed.map((line, index) => ({
          position: index + 1,
          side: line.side,
          account: line.account.trim(),
          amount: line.cents,
        })) as JournalLine[],
      });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-stone-900/40 flex items-start justify-center p-4 overflow-y-auto z-50">
      <form
        onSubmit={submit}
        className="bg-white rounded-2xl border border-stone-200 shadow-xl w-full max-w-3xl my-8"
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-stone-100">
          <h2 className="font-semibold text-stone-900">Neue Buchung</h2>
          <button type="button" onClick={onClose} className="text-stone-400 hover:text-stone-700">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <Field label="Buchungsdatum" hint="Bestimmt die Periode">
              <input
                type="date"
                value={bookingDate}
                onChange={(e) => setBookingDate(e.target.value)}
                className={inputClass}
                required
              />
            </Field>
            <Field label="Belegdatum" hint="Rechnungsdatum">
              <input
                type="date"
                value={documentDate}
                onChange={(e) => setDocumentDate(e.target.value)}
                className={inputClass}
                required
              />
            </Field>
            <Field label="Leistung von" hint="§ 14 Abs. 4 Nr. 6 UStG">
              <input
                type="date"
                value={serviceFrom}
                onChange={(e) => setServiceFrom(e.target.value)}
                className={inputClass}
                required
              />
            </Field>
            <Field label="Leistung bis">
              <input
                type="date"
                value={serviceTo}
                onChange={(e) => setServiceTo(e.target.value)}
                className={inputClass}
                required
              />
            </Field>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div className="sm:col-span-2">
              <Field label="Buchungstext">
                <input
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Wofür wurde gebucht?"
                  className={inputClass}
                  required
                />
              </Field>
            </div>
            <Field label="Belegnummer">
              <input
                value={documentNumber}
                onChange={(e) => setDocumentNumber(e.target.value)}
                placeholder="z. B. ER-2026-0042"
                className={inputClass}
              />
            </Field>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-stone-700 uppercase tracking-wide">
                Buchungszeilen
                <HelpTooltip
                  title="Warum mehr als zwei Zeilen?"
                  content={
                    'Ein Beleg mit Vorsteuer hat drei Zeilen: Aufwand und Vorsteuer im Soll, ' +
                    'die Verbindlichkeit im Haben. Bei Reverse Charge sind es vier. Die Summe der ' +
                    'Sollzeilen muss der Summe der Habenzeilen entsprechen.'
                  }
                />
              </span>
              <button
                type="button"
                onClick={() => setLines((prev) => [...prev, { side: 'S', account: '', amount: '' }])}
                className="text-xs text-amber-800 hover:text-amber-900 font-medium"
              >
                Zeile hinzufügen
              </button>
            </div>

            {lines.map((line, index) => (
              <div key={index} className="flex gap-2 items-center">
                <select
                  value={line.side}
                  onChange={(e) => updateLine(index, { side: e.target.value as Side })}
                  className={`${inputClass} w-24 shrink-0`}
                >
                  <option value="S">Soll</option>
                  <option value="H">Haben</option>
                </select>
                <input
                  list="postable-accounts"
                  value={line.account}
                  onChange={(e) => updateLine(index, { account: e.target.value })}
                  placeholder="Konto"
                  className={`${inputClass} w-32 shrink-0 font-mono`}
                  required
                />
                <span className="text-xs text-stone-500 truncate flex-1">
                  {postable.find((a) => a.number === line.account)?.name ?? ''}
                </span>
                <input
                  value={line.amount}
                  onChange={(e) => updateLine(index, { amount: e.target.value })}
                  placeholder="0,00"
                  inputMode="decimal"
                  className={`${inputClass} w-32 shrink-0 text-right font-mono`}
                  required
                />
                {lines.length > 2 && (
                  <button
                    type="button"
                    onClick={() => setLines((prev) => prev.filter((_, i) => i !== index))}
                    className="text-stone-400 hover:text-rose-600 shrink-0"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
              </div>
            ))}

            <datalist id="postable-accounts">
              {postable.map((a) => (
                <option key={a.number} value={a.number}>
                  {a.name}
                </option>
              ))}
            </datalist>

            <div
              className={`flex justify-between items-center text-sm rounded-lg px-3 py-2 border ${
                balanced
                  ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
                  : 'bg-stone-50 border-stone-200 text-stone-600'
              }`}
            >
              <span className="font-medium">
                {balanced
                  ? 'Soll und Haben stimmen überein'
                  : `Differenz ${formatCents(debitTotal - creditTotal)}`}
              </span>
              <span className="font-mono tabular-nums">
                Soll {formatCents(debitTotal)} · Haben {formatCents(creditTotal)}
              </span>
            </div>
          </div>

          {error && (
            <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-lg p-3">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 px-5 py-3 border-t border-stone-100">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-2 text-sm rounded-lg border border-stone-200 text-stone-700 hover:bg-stone-50"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            disabled={!balanced || saving}
            className="px-3 py-2 text-sm rounded-lg bg-amber-700 text-white hover:bg-amber-800 disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Wird gebucht…' : 'Buchen'}
          </button>
        </div>
      </form>
    </div>
  );
};

const inputClass =
  'w-full px-2.5 py-1.5 text-sm rounded-lg border border-stone-200 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none';

const Field: React.FC<{ label: string; hint?: string; children: React.ReactNode }> = ({
  label,
  hint,
  children,
}) => (
  <label className="block">
    <span className="block text-xs font-medium text-stone-600 mb-1">
      {label}
      {hint && <span className="text-stone-400 font-normal"> · {hint}</span>}
    </span>
    {children}
  </label>
);

// -------------------------------------------------------------------------

const ReverseDialog: React.FC<{
  entry: JournalEntry;
  onClose: () => void;
  onDone: () => void;
}> = ({ entry, onClose, onDone }) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await Api.reverseJournalEntry(entry.id, reason);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-stone-900/40 flex items-center justify-center p-4 z-50">
      <form onSubmit={submit} className="bg-white rounded-2xl border border-stone-200 shadow-xl w-full max-w-lg">
        <div className="px-5 py-3 border-b border-stone-100">
          <h2 className="font-semibold text-stone-900">Buchung stornieren</h2>
        </div>
        <div className="p-5 space-y-3">
          <p className="text-sm text-stone-600">
            Buchung <span className="font-mono">{entry.entryNumber}</span> über{' '}
            <span className="font-mono">
              {formatCents(entry.lines.filter((l) => l.side === 'S').reduce((s, l) => s + l.amount, 0))}
            </span>
            .
          </p>
          <p className="text-xs text-stone-500 bg-stone-50 border border-stone-200 rounded-lg p-3 leading-relaxed">
            Storniert wird per <strong>Generalumkehr</strong>: dieselben Konten auf denselben Seiten mit
            negiertem Betrag. Die Umsätze der betroffenen Konten gehen dadurch auf null zurück, statt sich
            wie bei einer spiegelverkehrten Gegenbuchung zu verdoppeln. Die Stornobuchung wird auf heute
            datiert; die ursprüngliche Buchung bleibt im Journal sichtbar.
          </p>
          <Field label="Grund der Stornierung">
            <input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="z. B. Beleg doppelt erfasst"
              className={inputClass}
              required
            />
          </Field>
          {error && (
            <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-lg p-3">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>
        <div className="flex justify-end gap-2 px-5 py-3 border-t border-stone-100">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-2 text-sm rounded-lg border border-stone-200 text-stone-700 hover:bg-stone-50"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            disabled={busy || !reason.trim()}
            className="px-3 py-2 text-sm rounded-lg bg-rose-700 text-white hover:bg-rose-800 disabled:opacity-40"
          >
            {busy ? 'Wird storniert…' : 'Stornieren'}
          </button>
        </div>
      </form>
    </div>
  );
};
