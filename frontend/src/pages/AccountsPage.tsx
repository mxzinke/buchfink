import React, { useEffect, useMemo, useState } from 'react';
import {
  AlertCircle,
  ArrowLeft,
  BookOpen,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  RefreshCw,
  Scale,
  Search,
} from 'lucide-react';
import type { Account, AccountLedger, AccountType, SuSaOverview } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

/**
 * Kontenaufstellung.
 *
 * Von 1.855 Katalogeinträgen bebucht ein Unternehmen ein paar Dutzend. Die
 * Seite zeigt deshalb zuerst die bebuchten Konten und macht den Kontenrahmen
 * erst auf Wunsch auf. Das Grundprinzip — Aktiv- und Aufwandskonten tragen
 * einen Sollsaldo, Passiv-, Kapital- und Ertragskonten einen Habensaldo —
 * steht direkt an den Zahlen statt in einer Legende.
 */

type Tab = 'konten' | 'susa';

const CLASS_NAMES: Record<number, string> = {
  0: 'Anlagevermögen',
  1: 'Umlaufvermögen',
  2: 'Eigen- & Fremdkapital',
  3: 'Fremdkapital',
  4: 'Betriebliche Erträge',
  5: 'Material & Fremdleistungen',
  6: 'Personal, Abschreibungen & sonstige Aufwendungen',
  7: 'Finanzen & Steuern',
  8: 'Von DATEV freigehalten',
  9: 'Vorträge & statistische Konten',
};

const TYPE_LABELS: Record<AccountType, { label: string; classes: string }> = {
  asset: { label: 'Aktiva', classes: 'bg-sky-50 text-sky-700 border-sky-200' },
  liability: { label: 'Passiva', classes: 'bg-rose-50 text-rose-700 border-rose-200' },
  equity: { label: 'Eigenkapital', classes: 'bg-violet-50 text-violet-700 border-violet-200' },
  revenue: { label: 'Ertrag', classes: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
  expense: { label: 'Aufwand', classes: 'bg-amber-50 text-amber-700 border-amber-200' },
  statistical: { label: 'Statistisch', classes: 'bg-stone-100 text-stone-600 border-stone-200' },
};

/** Auf welcher Seite ein Konto seinen normalen Saldo trägt. */
function naturalSide(type: AccountType): 'Soll' | 'Haben' {
  return type === 'liability' || type === 'equity' || type === 'revenue' ? 'Haben' : 'Soll';
}

/** Beschreibt den Saldo eines Kontos ausgeschrieben. */
function balanceHint(account: Account): string {
  const natural = naturalSide(account.type);
  if (account.balance === 0) return 'ausgeglichen';
  if (account.balance > 0) return `${natural}saldo`;
  return `${natural === 'Soll' ? 'Haben' : 'Soll'}saldo (ungewöhnlich für dieses Konto)`;
}

const TypeBadge: React.FC<{ type: AccountType }> = ({ type }) => {
  const meta = TYPE_LABELS[type] ?? TYPE_LABELS.asset;
  return (
    <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded border ${meta.classes}`}>
      {meta.label}
    </span>
  );
};

export const AccountsPage: React.FC = () => {
  const [tab, setTab] = useState<Tab>('konten');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [susa, setSusa] = useState<SuSaOverview | null>(null);
  const [ledger, setLedger] = useState<AccountLedger | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingLedger, setLoadingLedger] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [showCatalog, setShowCatalog] = useState(false);
  const [openClasses, setOpenClasses] = useState<Record<number, boolean>>({});

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [accountList, overview] = await Promise.all([Api.getAccounts(), Api.getSuSaOverview()]);
      setAccounts(accountList);
      setSusa(overview);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function openLedger(accountNumber: string) {
    setLoadingLedger(true);
    setError(null);
    try {
      setLedger(await Api.getAccountLedger(accountNumber));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoadingLedger(false);
    }
  }

  /** Bebuchte Konten — das, womit tatsächlich gearbeitet wird. */
  const inUse = useMemo(
    () => accounts.filter((a) => a.bookingsCount > 0).sort((a, b) => a.number.localeCompare(b.number)),
    [accounts]
  );

  /** Bebuchbarer Katalog: ohne reservierte Einträge und ohne die freigehaltene Klasse 8. */
  const catalog = useMemo(
    () => accounts.filter((a) => !a.isReserved && a.kontenklasse !== 8),
    [accounts]
  );

  const searchResults = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return [];
    return catalog
      .filter((a) => a.number.toLowerCase().includes(query) || a.name.toLowerCase().includes(query))
      .slice(0, 60);
  }, [catalog, search]);

  const catalogByClass = useMemo(() => {
    const groups = new Map<number, Account[]>();
    for (const account of catalog) {
      const list = groups.get(account.kontenklasse) ?? [];
      list.push(account);
      groups.set(account.kontenklasse, list);
    }
    return [...groups.entries()]
      .sort(([a], [b]) => a - b)
      .map(([kontenklasse, list]) => ({
        kontenklasse,
        accounts: list.sort((a, b) => a.number.localeCompare(b.number)),
      }));
  }, [catalog]);

  if (ledger) {
    return (
      <LedgerView
        ledger={ledger}
        loading={loadingLedger}
        onBack={() => setLedger(null)}
      />
    );
  }

  return (
    <div className="p-4 sm:p-6 space-y-5 max-w-6xl mx-auto">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold text-stone-900 tracking-tight">Konten</h1>
        <p className="text-sm text-stone-600">
          Kontenrahmen SKR04 der DATEV, Fassung 2026. Gebucht wird auf Sachkonten mit vier Stellen;
          Personenkonten für Kunden und Lieferanten liegen in eigenen Nummernkreisen.
        </p>
      </header>

      {error && (
        <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-xl p-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <nav className="flex gap-1 border-b border-stone-200">
        {(
          [
            { id: 'konten', label: 'Konten', icon: BookOpen },
            { id: 'susa', label: 'Summen & Salden', icon: Scale },
          ] as const
        ).map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setTab(id)}
            className={`flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
              tab === id
                ? 'border-amber-600 text-amber-800'
                : 'border-transparent text-stone-500 hover:text-stone-800'
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </nav>

      {loading ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-stone-400">
          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
          Konten werden geladen…
        </div>
      ) : tab === 'konten' ? (
        <>
          <section className="bg-white rounded-2xl border border-stone-200/80 shadow-xs overflow-hidden">
            <div className="px-4 py-3 border-b border-stone-100 flex items-baseline justify-between gap-3">
              <h2 className="text-sm font-semibold text-stone-800">
                Bebuchte Konten
                <HelpTooltip
                  title="Bebuchte Konten"
                  content="Der SKR04 enthält über 1.600 nutzbare Konten. Ein Unternehmen bebucht davon typischerweise ein paar Dutzend. Diese Liste zeigt genau die."
                  tip="Neue Konten entstehen automatisch, sobald eine Buchungsgruppe sie zum ersten Mal verwendet."
                />
              </h2>
              <span className="text-xs text-stone-500 tabular-nums">
                {inUse.length} von {catalog.length} bebuchbaren
              </span>
            </div>

            {inUse.length === 0 ? (
              <p className="px-4 py-8 text-sm text-stone-500 text-center">
                Noch keine Buchungen vorhanden. Sobald der erste Beleg erfasst ist, erscheinen hier die
                Konten, die er berührt.
              </p>
            ) : (
              <AccountTable accounts={inUse} onSelect={openLedger} showTurnover />
            )}
          </section>

          <section className="bg-white rounded-2xl border border-stone-200/80 shadow-xs overflow-hidden">
            <div className="px-4 py-3 border-b border-stone-100 space-y-3">
              <div className="flex items-baseline justify-between gap-3">
                <h2 className="text-sm font-semibold text-stone-800">Kontenrahmen durchsuchen</h2>
                <button
                  onClick={() => setShowCatalog((v) => !v)}
                  className="text-xs text-amber-800 hover:text-amber-900 font-medium"
                >
                  {showCatalog ? 'Vollständigen Rahmen ausblenden' : 'Vollständigen Rahmen anzeigen'}
                </button>
              </div>
              <div className="relative">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-stone-400" />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Kontonummer oder Bezeichnung, z. B. 6815 oder Bürobedarf"
                  className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-stone-200 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none"
                />
              </div>
              <p className="text-xs text-stone-500 leading-relaxed">
                Bereichskonten wie <span className="font-mono">4400-4409</span> sind eine
                Kurzschreibweise für zehn nutzbare Konten, keine eigenen Konten — gebucht wird auf 4400
                bis 4409. Kontenklasse 8 hält die DATEV im SKR04 frei; sie ist hier ausgeblendet, weil
                dort nicht gebucht werden darf.
              </p>
            </div>

            {search.trim() ? (
              searchResults.length ? (
                <AccountTable accounts={searchResults} onSelect={openLedger} />
              ) : (
                <p className="px-4 py-8 text-sm text-stone-500 text-center">
                  Kein Konto gefunden. Beachte: Erlöse liegen im SKR04 in Klasse 4 (z. B. 4400), nicht in
                  Klasse 8 wie im SKR03.
                </p>
              )
            ) : showCatalog ? (
              <div className="divide-y divide-stone-100">
                {catalogByClass.map(({ kontenklasse, accounts: classAccounts }) => {
                  const isOpen = openClasses[kontenklasse] ?? false;
                  return (
                    <div key={kontenklasse}>
                      <button
                        onClick={() =>
                          setOpenClasses((prev) => ({ ...prev, [kontenklasse]: !prev[kontenklasse] }))
                        }
                        className="w-full flex items-center gap-2 px-4 py-2.5 hover:bg-stone-50 text-left"
                      >
                        {isOpen ? (
                          <ChevronDown className="w-4 h-4 text-stone-400" />
                        ) : (
                          <ChevronRight className="w-4 h-4 text-stone-400" />
                        )}
                        <span className="font-mono text-xs text-stone-500">Klasse {kontenklasse}</span>
                        <span className="text-sm font-medium text-stone-800">
                          {CLASS_NAMES[kontenklasse] ?? ''}
                        </span>
                        <span className="ml-auto text-xs text-stone-400 tabular-nums">
                          {classAccounts.length}
                        </span>
                      </button>
                      {isOpen && <AccountTable accounts={classAccounts} onSelect={openLedger} />}
                    </div>
                  );
                })}
              </div>
            ) : null}
          </section>
        </>
      ) : (
        <SuSaView susa={susa} onSelect={openLedger} />
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

const AccountTable: React.FC<{
  accounts: Account[];
  onSelect: (accountNumber: string) => void;
  showTurnover?: boolean;
}> = ({ accounts, onSelect, showTurnover = false }) => (
  <table className="w-full text-sm">
    <thead>
      <tr className="text-[11px] uppercase tracking-wide text-stone-400 border-b border-stone-100">
        <th className="text-left font-medium px-4 py-2">Konto</th>
        <th className="text-left font-medium px-2 py-2">Bezeichnung</th>
        {showTurnover && (
          <>
            <th className="text-right font-medium px-2 py-2">Soll</th>
            <th className="text-right font-medium px-2 py-2">Haben</th>
          </>
        )}
        <th className="text-right font-medium px-4 py-2">Saldo</th>
      </tr>
    </thead>
    <tbody className="divide-y divide-stone-50">
      {accounts.map((account) => {
        const postable = !account.isRange;
        return (
          <tr
            key={account.number}
            onClick={() => postable && onSelect(account.number)}
            className={postable ? 'hover:bg-amber-50/40 cursor-pointer' : 'bg-stone-50/50'}
          >
            <td className="px-4 py-2 font-mono text-xs text-stone-700 whitespace-nowrap">
              {account.number}
            </td>
            <td className="px-2 py-2">
              <div className="flex items-center gap-2">
                <span className="text-stone-800">{account.name}</span>
                <TypeBadge type={account.type} />
                {account.isRange && (
                  <span
                    className="text-[10px] text-stone-500 border border-stone-200 rounded px-1.5 py-0.5"
                    title="Kurzschreibweise für die Konten dieses Bereichs — nicht selbst bebuchbar"
                  >
                    Bereich
                  </span>
                )}
              </div>
            </td>
            {showTurnover && (
              <>
                <td className="px-2 py-2 text-right font-mono text-xs text-stone-500 tabular-nums">
                  {account.debitSum ? formatCents(account.debitSum) : '—'}
                </td>
                <td className="px-2 py-2 text-right font-mono text-xs text-stone-500 tabular-nums">
                  {account.creditSum ? formatCents(account.creditSum) : '—'}
                </td>
              </>
            )}
            <td className="px-4 py-2 text-right whitespace-nowrap">
              <div
                className={`font-mono text-sm tabular-nums ${
                  account.balance < 0 ? 'text-rose-600' : 'text-stone-900'
                }`}
              >
                {formatCents(account.balance)}
              </div>
              {account.bookingsCount > 0 && (
                <div className="text-[10px] text-stone-400">{balanceHint(account)}</div>
              )}
            </td>
          </tr>
        );
      })}
    </tbody>
  </table>
);

// -------------------------------------------------------------------------

const SuSaView: React.FC<{ susa: SuSaOverview | null; onSelect: (n: string) => void }> = ({
  susa,
  onSelect,
}) => {
  if (!susa) return null;

  const classesWithBookings = susa.classes.filter((c) => c.accountsCount > 0);

  return (
    <div className="space-y-4">
      <div
        className={`flex items-start gap-2 rounded-xl p-3 text-sm border ${
          susa.isBalanced
            ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
            : 'bg-rose-50 border-rose-200 text-rose-800'
        }`}
      >
        {susa.isBalanced ? (
          <CheckCircle2 className="w-4 h-4 mt-0.5 shrink-0" />
        ) : (
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
        )}
        <div>
          {susa.isBalanced ? (
            <>
              <span className="font-medium">Soll und Haben stimmen überein.</span> Beide Seiten stehen bei{' '}
              <span className="font-mono">{formatCents(susa.totalDebit)}</span>. Die Prüfung ist exakt,
              nicht auf Cent gerundet.
            </>
          ) : (
            <>
              <span className="font-medium">
                Soll und Haben weichen um {formatCents(susa.difference)} ab.
              </span>{' '}
              Das darf nicht vorkommen — jede Buchung wird beim Speichern auf Ausgeglichenheit geprüft.
              Bitte die Integritätsprüfung im Journal ausführen.
            </>
          )}
        </div>
      </div>

      {classesWithBookings.length === 0 ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-sm text-stone-500">
          Noch keine Buchungen im aktiven Geschäftsjahr.
        </div>
      ) : (
        classesWithBookings.map((cls) => (
          <section
            key={cls.kontenklasse}
            className="bg-white rounded-2xl border border-stone-200/80 shadow-xs overflow-hidden"
          >
            <div className="px-4 py-2.5 border-b border-stone-100 flex items-baseline gap-2">
              <span className="font-mono text-xs text-stone-500">Klasse {cls.kontenklasse}</span>
              <h3 className="text-sm font-semibold text-stone-800">
                {CLASS_NAMES[cls.kontenklasse] ?? cls.kontenklasseName}
              </h3>
              <span className="ml-auto text-xs text-stone-500 tabular-nums">
                Soll {formatCents(cls.totalDebit)} · Haben {formatCents(cls.totalCredit)}
              </span>
            </div>
            <AccountTable accounts={cls.accounts} onSelect={onSelect} showTurnover />
          </section>
        ))
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

const LedgerView: React.FC<{
  ledger: AccountLedger;
  loading: boolean;
  onBack: () => void;
}> = ({ ledger, loading, onBack }) => {
  const account = ledger.account;
  const rows = ledger.rows ?? [];

  return (
    <div className="p-4 sm:p-6 space-y-4 max-w-6xl mx-auto">
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-stone-600 hover:text-stone-900"
      >
        <ArrowLeft className="w-4 h-4" />
        Zurück zur Kontenübersicht
      </button>

      <header className="bg-white p-4 sm:p-5 rounded-2xl border border-stone-200/80 shadow-xs">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-1.5">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xl font-mono font-bold text-amber-800 bg-amber-50 px-2.5 py-0.5 rounded-lg border border-amber-200/60">
                {account.number}
              </span>
              <h1 className="text-xl font-bold text-stone-900 tracking-tight">{account.name}</h1>
              <TypeBadge type={account.type} />
            </div>
            <p className="text-xs text-stone-500">
              Kontenklasse {account.kontenklasse} · {account.posten || account.category}
            </p>
          </div>

          <div className="bg-stone-50 px-4 py-3 rounded-xl border border-stone-200/60 text-right">
            <div className="text-[11px] font-medium text-stone-500 uppercase tracking-wider">
              Saldo im GJ {ledger.fiscalYear}
            </div>
            <div
              className={`text-2xl font-mono font-bold tabular-nums ${
                ledger.closingBalance < 0 ? 'text-rose-600' : 'text-stone-900'
              }`}
            >
              {formatCents(ledger.closingBalance)}
            </div>
            <div className="text-[10px] text-stone-500 mt-0.5">{balanceHint(account)}</div>
          </div>
        </div>

        <div className="mt-4 pt-3 border-t border-stone-100 flex flex-wrap gap-x-8 gap-y-1 text-xs text-stone-600">
          <span>
            Summe Soll <span className="font-mono text-stone-900">{formatCents(ledger.totalDebit)}</span>
          </span>
          <span>
            Summe Haben{' '}
            <span className="font-mono text-stone-900">{formatCents(ledger.totalCredit)}</span>
          </span>
          <span>
            Zeilen <span className="font-mono text-stone-900">{ledger.rowCount}</span>
          </span>
        </div>
      </header>

      {loading ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-stone-400">
          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
          Kontoblatt wird geladen…
        </div>
      ) : rows.length === 0 ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-sm text-stone-500">
          Auf diesem Konto wurde im Geschäftsjahr {ledger.fiscalYear} nicht gebucht.
        </div>
      ) : (
        <div className="bg-white rounded-2xl border border-stone-200/80 shadow-xs overflow-x-auto">
          <table className="w-full text-sm min-w-[52rem]">
            <thead>
              <tr className="text-[11px] uppercase tracking-wide text-stone-400 border-b border-stone-100">
                <th className="text-left font-medium px-4 py-2">Datum</th>
                <th className="text-left font-medium px-2 py-2">Buchung</th>
                <th className="text-left font-medium px-2 py-2">Text</th>
                <th className="text-left font-medium px-2 py-2">
                  Gegenkonten
                  <HelpTooltip
                    title="Mehrere Gegenkonten"
                    content={
                      'Eine Buchung besteht aus beliebig vielen Zeilen. „Aufwand und Vorsteuer an ' +
                      'Verbindlichkeit“ hat drei, ein Reverse-Charge-Vorgang vier. Deshalb steht hier ' +
                      'eine Liste und nicht ein einzelnes Gegenkonto.'
                    }
                  />
                </th>
                <th className="text-right font-medium px-2 py-2">Soll</th>
                <th className="text-right font-medium px-2 py-2">Haben</th>
                <th className="text-right font-medium px-4 py-2">Saldo</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-50">
              {rows.map((row, index) => (
                <tr
                  key={`${row.entryId}-${index}`}
                  className={row.kind === 'reversal' ? 'bg-rose-50/40' : undefined}
                >
                  <td className="px-4 py-2 whitespace-nowrap text-xs text-stone-600">
                    {formatDate(row.bookingDate)}
                  </td>
                  <td className="px-2 py-2 whitespace-nowrap">
                    <div className="font-mono text-xs text-stone-700">{row.entryNumber}</div>
                    {row.kind === 'reversal' && (
                      <span className="text-[10px] text-rose-700 font-medium">Generalumkehr</span>
                    )}
                  </td>
                  <td className="px-2 py-2 text-stone-800">
                    <div>{row.description}</div>
                    {row.documentNumber && (
                      <div className="text-[10px] text-stone-400 font-mono">{row.documentNumber}</div>
                    )}
                  </td>
                  <td className="px-2 py-2">
                    <div className="flex flex-wrap gap-1">
                      {(row.counterAccounts ?? []).map((counter, i) => (
                        <span
                          key={`${counter.account}-${i}`}
                          className="text-[10px] font-mono bg-stone-100 text-stone-600 rounded px-1.5 py-0.5"
                          title={`${counter.name} · ${formatCents(counter.amount)}`}
                        >
                          {counter.account}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-2 py-2 text-right font-mono text-xs tabular-nums text-stone-700">
                    {row.debitAmount ? formatCents(row.debitAmount) : ''}
                  </td>
                  <td className="px-2 py-2 text-right font-mono text-xs tabular-nums text-stone-700">
                    {row.creditAmount ? formatCents(row.creditAmount) : ''}
                  </td>
                  <td className="px-4 py-2 text-right font-mono text-xs tabular-nums font-medium text-stone-900">
                    {formatCents(row.runningBalance)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
