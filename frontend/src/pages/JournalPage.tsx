import React, { useEffect, useMemo, useState } from 'react';
import {
  AlertCircle,
  ChevronDown,
  ChevronRight,
  FileDown,
  Plus,
  ShieldCheck,
  Trash2,
  Undo2,
} from 'lucide-react';
import type {
  Account,
  JournalEntry,
  JournalLine,
  PaymentAllocationDetail,
  Side,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import {
  formatCents,
  formatDate,
  formatDateRange,
  formatShortHash,
  parseCents,
} from '../utils/formatters';
import {
  Button,
  Combobox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  PageHeader,
  SearchInput,
  Select,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

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

/** Bruttobetrag einer Buchung, gemessen an der Sollseite. */
function grossOf(entry: JournalEntry): number {
  return entry.lines.filter((line) => line.side === 'S').reduce((sum, line) => sum + line.amount, 0);
}

/** Konten in der Schreibweise Soll → Haben, so wie sie im Journal steht. */
function accountPath(entry: JournalEntry): string {
  const debit = entry.lines.filter((l) => l.side === 'S').map((l) => l.account);
  const credit = entry.lines.filter((l) => l.side === 'H').map((l) => l.account);
  return `${debit.join(' · ')} → ${credit.join(' · ')}`;
}

export interface JournalPageProps {
  /**
   * Das angezeigte Geschäftsjahr, sofern es festgestellt oder offengelegt ist.
   * Es nimmt keine Buchung mehr an; die Erfassung bleibt an ihrem Platz und
   * wird deaktiviert, damit die Ansicht zwischen den Jahren gleich aussieht
   * (§11.5). Grund und Hinweisstreifen der Sperre stehen zentral in App.tsx,
   * damit jede Erfassungsansicht sie zeigt und keine sie vergisst.
   */
  closedYear?: number;
  /**
   * Buchungsnummer aus dem Navigationsziel. Sie steht als Suchbegriff im Feld
   * und nicht als versteckter Filter: der Weg vom Kontoblatt zur Buchung
   * (GOB-02) endet in einer Liste, die man weiter durchsuchen können muss.
   */
  initialSearch?: string;
}

export const JournalPage: React.FC<JournalPageProps> = ({ closedYear, initialSearch }) => {
  // Zwei Sperren mit demselben Ergebnis: das festgestellte Geschäftsjahr und
  // der Prüfermodus. Beide gehören in den title des Knopfes, damit der Grund
  // nicht in der Fehlermeldung des ersten Versuchs steht (§10.4).
  const writeLock = useWriteLock();
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [integrityError, setIntegrityError] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);
  const [search, setSearch] = useState(initialSearch ?? '');
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});
  // Der Zeitraum ist die Frage des Prüfers: er fragt nach Datumsgrenzen und
  // nicht nach einem Geschäftsjahr (BEL-05). Sobald eine Grenze steht, liest
  // die Seite alle Jahre — sonst zeigte ein Fenster über den Jahreswechsel nur
  // dessen zweite Hälfte.
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [exportingCsv, setExportingCsv] = useState(false);

  const [showForm, setShowForm] = useState(false);
  const [reversing, setReversing] = useState<JournalEntry | null>(null);

  useEffect(() => {
    void load();
  }, [from, to]);

  // Der Suchbegriff folgt dem Navigationsziel, auch wenn die Seite schon steht:
  // wer nach dem Weg vom Kontoblatt zur Buchung erneut „Journal" wählt, will die
  // ganze Liste sehen und nicht weiter den Filter auf eine Buchungsnummer.
  useEffect(() => {
    setSearch(initialSearch ?? '');
  }, [initialSearch]);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const ranged = Boolean(from || to);
      const [entryList, accountList] = await Promise.all([
        ranged ? Api.getAllJournalEntries() : Api.getJournalEntries(),
        Api.getAccounts(),
      ]);
      setEntries(entryList);
      setAccounts(accountList);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  /**
   * Der dauerhafte Zustand steht im Fuß der Navigation (§11.4). Hier zählt nur
   * das Ergebnis der ausgelösten Prüfung: gelungen als Toast, ein Bruch als
   * Hinweisfläche, die stehen bleibt.
   */
  async function runIntegrityCheck() {
    setChecking(true);
    setIntegrityError(null);
    try {
      const result = await Api.verifyIntegrity();
      if (result.isValid) {
        toast.success(`Kette geprüft, ${result.checkedEntries} Buchungen unverändert.`);
      } else {
        setIntegrityError(
          `${result.message} Kettenkopf ${formatShortHash(result.lastVerifiedHash)}.`,
        );
      }
    } catch (e) {
      setIntegrityError(e instanceof Error ? e.message : String(e));
    } finally {
      setChecking(false);
    }
  }

  const accountNames = useMemo(() => {
    const map = new Map<string, string>();
    for (const account of accounts) map.set(account.number, account.name);
    return map;
  }, [accounts]);

  /** Ursprungsbuchungen, zu denen es eine Generalumkehr gibt. */
  const reversedIds = useMemo(() => {
    const ids = new Set<number>();
    for (const entry of entries) if (entry.reversalOfId) ids.add(entry.reversalOfId);
    return ids;
  }, [entries]);

  const entryNumbers = useMemo(() => {
    const map = new Map<number, string>();
    for (const entry of entries) map.set(entry.id, entry.entryNumber);
    return map;
  }, [entries]);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    // Die Datumsgrenzen sind ISO-Zeichenketten und lassen sich deshalb
    // vergleichen, ohne sie in ein Datum zu wandeln.
    const inRange = (entry: JournalEntry) =>
      (!from || entry.bookingDate >= from) && (!to || entry.bookingDate <= to);
    return entries.filter(
      (entry) =>
        inRange(entry) &&
        (!query ||
          entry.entryNumber.toLowerCase().includes(query) ||
          entry.description.toLowerCase().includes(query) ||
          (entry.documentNumber ?? '').toLowerCase().includes(query) ||
          entry.lines.some((line) => line.account.includes(query))),
    );
  }, [entries, search, from, to]);

  /** Das Journal des gewählten Zeitraums als CSV — dieselbe Tabelle wie im Z3-Export. */
  async function exportCsv() {
    setExportingCsv(true);
    try {
      const path = await Api.exportJournalCSV(from, to);
      // Leerer Pfad heißt: der Speichern-Dialog wurde abgebrochen.
      if (path) toast.success(`Journal gespeichert: ${path}`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setExportingCsv(false);
    }
  }

  const fiscalYear = entries[0]?.fiscalYear;

  // Der Grund der Sperre steht im title des deaktivierten Knopfes, sonst
  // verschwiege er ihn (§10.4).
  const lockedReason =
    writeLock.hint ??
    (closedYear
      ? `Das Geschäftsjahr ${closedYear} ist festgestellt. Buchungen nimmt es erst wieder an, ` +
        `wenn die Feststellung mit Grund zurückgesetzt wird.`
      : undefined);
  const postingLocked = writeLock.locked || Boolean(closedYear);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Journal"
        context={
          // Mit Zeitraum steht die Zahl der gefilterten Buchungen da: die Liste
          // umfasst dann alle Jahre, und „Geschäftsjahr 2026" wäre falsch.
          from || to
            ? `${filtered.length} Buchungen · ${from ? formatDate(from) : 'Anfang'} bis ${to ? formatDate(to) : 'heute'}`
            : fiscalYear
              ? `${entries.length} Buchungen · Geschäftsjahr ${fiscalYear} · lokal gespeichert`
              : 'Lokal gespeichert'
        }
        action={
          <Button
            variant="primary"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            disabled={postingLocked}
            title={lockedReason}
            onClick={() => setShowForm(true)}
          >
            Neue Buchung
          </Button>
        }
      />

      {/* Ein Integritätsbruch ist der einzige Fall, in dem die Oberfläche laut
          werden darf (§11.4). */}
      {integrityError && (
        <div className="mt-6 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
          <p className="text-body text-negative-text flex-1">{integrityError}</p>
          <Button variant="quiet" size="sm" onClick={() => setIntegrityError(null)}>
            Verstanden
          </Button>
        </div>
      )}

      {error && (
        <div className="mt-6 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
          <p className="text-body text-negative-text">{error}</p>
        </div>
      )}

      <div className="mt-6 flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Belegnummer, Buchungstext oder Konto"
          className="max-w-md"
        />
        <Input
          type="date"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
          aria-label="Buchungen ab"
          title="Buchungen ab diesem Buchungsdatum"
          className="w-40"
        />
        <Input
          type="date"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          aria-label="Buchungen bis"
          title="Buchungen bis zu diesem Buchungsdatum"
          className="w-40"
        />
        <Button
          variant="secondary"
          loading={exportingCsv}
          onClick={() => void exportCsv()}
          icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
        >
          Als CSV
        </Button>
        <Button
          variant="secondary"
          loading={checking}
          onClick={runIntegrityCheck}
          icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
        >
          Integrität prüfen
        </Button>
      </div>

      <div className="mt-5">
        {loading ? (
          <SkeletonRows rows={8} />
        ) : filtered.length === 0 ? (
          <EmptyState
            variant={entries.length === 0 ? 'leer' : 'gefiltert'}
            title={
              entries.length === 0
                ? 'Noch keine Buchungen erfasst'
                : 'Keine Buchung passt zur Suche'
            }
            description={
              entries.length === 0
                ? 'Buchungen entstehen aus dem Abgleich von Bankumsätzen mit Belegen oder direkt hier im Journal.'
                : undefined
            }
            action={
              entries.length === 0 ? (
                <Button
                  variant="primary"
                  disabled={postingLocked}
                  title={lockedReason}
                  onClick={() => setShowForm(true)}
                >
                  Neue Buchung
                </Button>
              ) : (
                <Button variant="secondary" onClick={() => setSearch('')}>
                  Suche zurücksetzen
                </Button>
              )
            }
          />
        ) : (
          <Table>
            <Thead sticky>
              <Tr>
                <Th className="w-8" aria-label="Aufklappen" />
                <Th>Beleg</Th>
                <Th>Datum</Th>
                <Th>Buchungstext</Th>
                <Th>Konten · Soll → Haben</Th>
                <Th numeric>Betrag</Th>
                <Th>Status</Th>
                <Th className="w-10" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {filtered.map((entry) => (
                <EntryRows
                  key={entry.id}
                  entry={entry}
                  accountNames={accountNames}
                  originNumber={
                    entry.reversalOfId ? entryNumbers.get(entry.reversalOfId) : undefined
                  }
                  isReversed={reversedIds.has(entry.id)}
                  expanded={expanded[entry.id] ?? false}
                  onToggle={() =>
                    setExpanded((prev) => ({ ...prev, [entry.id]: !prev[entry.id] }))
                  }
                  onReverse={() => setReversing(entry)}
                />
              ))}
            </Tbody>
          </Table>
        )}
      </div>

      <BookingForm
        open={showForm}
        accounts={accounts}
        onOpenChange={setShowForm}
        onSaved={async () => {
          setShowForm(false);
          await load();
          toast.success('Buchung festgeschrieben.');
        }}
      />

      <ReverseDialog
        entry={reversing}
        onClose={() => setReversing(null)}
        onDone={async () => {
          setReversing(null);
          await load();
          toast.success('Generalumkehr gebucht.');
        }}
      />
    </div>
  );
};

// -------------------------------------------------------------------------

/**
 * Eine Buchung als Zeile, aufgeklappt zusätzlich der Buchungssatz.
 *
 * Storniert wird nichts durchgestrichen: Ursprung und Generalumkehr bleiben
 * beide lesbar, markiert durch Tönung, Pille und Badge (§11.2).
 */
const EntryRows: React.FC<{
  entry: JournalEntry;
  accountNames: Map<string, string>;
  originNumber?: string;
  isReversed: boolean;
  expanded: boolean;
  onToggle: () => void;
  onReverse: () => void;
}> = ({ entry, accountNames, originNumber, isReversed, expanded, onToggle, onReverse }) => {
  const writeLock = useWriteLock();
  const isReversal = entry.kind === 'reversal';
  const storno = isReversal || isReversed;

  return (
    <>
      <Tr variant={storno ? 'storno' : 'default'} className="group">
        <Td className="pr-0">
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={expanded}
            aria-label={expanded ? 'Buchungssatz einklappen' : 'Buchungssatz anzeigen'}
            className="grid place-items-center w-6 h-6 rounded-control text-ink-faint
                       transition-colors duration-120 ease-quiet hover:bg-sunken hover:text-ink"
          >
            {expanded ? (
              <ChevronDown className="w-4 h-4" strokeWidth={1.5} />
            ) : (
              <ChevronRight className="w-4 h-4" strokeWidth={1.5} />
            )}
          </button>
        </Td>
        <Td code>{entry.entryNumber}</Td>
        <Td className="text-ink-subtle num">{formatDate(entry.bookingDate)}</Td>
        <Td className="max-w-[22rem] truncate" title={entry.description}>
          {entry.description}
        </Td>
        <Td code>{accountPath(entry)}</Td>
        <Td numeric>{formatCents(grossOf(entry), entry.currency)}</Td>
        <Td>
          <StatusBadge status={storno ? 'storniert' : 'gebucht'} />
        </Td>
        <Td className="pl-0">
          {!isReversal && !isReversed && (
            <Button
              variant="quiet"
              size="sm"
              iconOnly
              disabled={writeLock.locked}
              onClick={onReverse}
              title={writeLock.hint ?? 'Buchung per Generalumkehr stornieren'}
              aria-label={`Buchung ${entry.entryNumber} stornieren`}
              className="opacity-0 transition-opacity duration-120 ease-quiet
                         group-hover:opacity-100 focus-visible:opacity-100"
            >
              <Undo2 className="w-4 h-4" strokeWidth={1.5} />
            </Button>
          )}
        </Td>
      </Tr>

      {expanded && (
        <Tr className={cn(storno && '[&>td]:bg-negative-soft/40')}>
          <Td colSpan={8} className="whitespace-normal py-4">
            <EntryDetail
              entry={entry}
              accountNames={accountNames}
              originNumber={originNumber}
            />
          </Td>
        </Tr>
      )}
    </>
  );
};

/** Der Buchungssatz: Soll links, Haben rechts, Summe mit Doppellinie (§11.1). */
const EntryDetail: React.FC<{
  entry: JournalEntry;
  accountNames: Map<string, string>;
  originNumber?: string;
}> = ({ entry, accountNames, originNumber }) => {
  const debit = entry.lines.filter((l) => l.side === 'S').reduce((s, l) => s + l.amount, 0);
  const credit = entry.lines.filter((l) => l.side === 'H').reduce((s, l) => s + l.amount, 0);

  return (
    <div className="pl-8 pr-2">
      <table className="w-full text-body">
        <thead>
          <tr className="[&>th]:h-7 [&>th]:text-label [&>th]:font-medium [&>th]:text-ink-subtle [&>th]:border-b [&>th]:border-line-strong">
            <th className="text-left w-24">Konto</th>
            <th className="text-left">Bezeichnung</th>
            <th className="text-right w-36">Soll</th>
            <th className="text-right w-36">Haben</th>
          </tr>
        </thead>
        <tbody>
          {entry.lines.map((line) => (
            <tr key={line.id || line.position} className="[&>td]:h-8 [&>td]:border-b [&>td]:border-line">
              <td className="code-num text-caption text-ink-muted">{line.account}</td>
              <td className="text-ink">
                {line.accountName || accountNames.get(line.account) || '—'}
                {line.taxKey && (
                  <span className="ml-2 code-num text-caption text-ink-subtle">{line.taxKey}</span>
                )}
              </td>
              <td className="text-right num">
                {line.side === 'S' ? formatCents(line.amount, entry.currency) : <span className="text-ink-subtle">—</span>}
              </td>
              <td className="text-right num">
                {line.side === 'H' ? formatCents(line.amount, entry.currency) : <span className="text-ink-subtle">—</span>}
              </td>
            </tr>
          ))}
          <tr className="[&>td]:h-9 [&>td]:rule-total [&>td]:font-semibold">
            <td />
            <td>Summe</td>
            <td className="text-right num">{formatCents(debit, entry.currency)}</td>
            <td className="text-right num">{formatCents(credit, entry.currency)}</td>
          </tr>
        </tbody>
      </table>

      <dl className="mt-4 flex flex-wrap gap-x-8 gap-y-1 text-caption text-ink-subtle">
        <div className="flex gap-1.5">
          <dt>Herkunft</dt>
          <dd className="text-ink-muted">{SOURCE_LABELS[entry.source] ?? entry.source}</dd>
        </div>
        <div className="flex gap-1.5">
          <dt>Beleg</dt>
          <dd className="text-ink-muted num">
            {formatDate(entry.documentDate)}
            {entry.documentNumber ? ` · ${entry.documentNumber}` : ''}
          </dd>
        </div>
        <div className="flex gap-1.5">
          <dt>Leistung</dt>
          <dd className="text-ink-muted num">
            {formatDateRange(entry.serviceDateFrom, entry.serviceDateTo)}
          </dd>
        </div>
        {originNumber && (
          <div className="flex gap-1.5">
            <dt>Storno zu</dt>
            <dd className="code-num text-negative-text">{originNumber}</dd>
          </div>
        )}
        {entry.reversalReason && (
          <div className="flex gap-1.5">
            <dt>Grund</dt>
            <dd className="text-ink-muted">{entry.reversalReason}</dd>
          </div>
        )}
        <div className="flex gap-1.5">
          <dt>Hash</dt>
          <dd className="font-mono text-ink-muted">
            {formatShortHash(entry.entryHash)} ← {formatShortHash(entry.previousHash)}
          </dd>
        </div>
      </dl>

      {entry.source === 'payment' && <PaymentAllocations entryId={entry.id} currency={entry.currency} />}
    </div>
  );
};

/**
 * Die Einzelposten einer Zahlungsbuchung.
 *
 * Eine Sammelüberweisung ist im Journal eine Zeile über einen runden Betrag;
 * wogegen sie lief, steht nur in den Zuordnungen. GoBD Rz. 36 verlangt, dass
 * sich jeder Geschäftsvorfall in seine Bestandteile zerlegen lässt — deshalb
 * werden sie erst beim Aufklappen geladen und dann vollständig gezeigt.
 */
const PaymentAllocations: React.FC<{ entryId: number; currency: string }> = ({
  entryId,
  currency,
}) => {
  const [allocations, setAllocations] = useState<PaymentAllocationDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Api.getPaymentAllocations(entryId)
      .then((list) => {
        if (!cancelled) setAllocations(list);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [entryId]);

  if (loading) return <SkeletonRows rows={2} className="mt-4" />;
  if (error) return <p className="mt-4 text-caption text-negative-text">{error}</p>;
  if (allocations.length === 0) {
    return (
      <p className="mt-4 text-caption text-ink-subtle">
        Diese Zahlung ist keinem offenen Posten zugeordnet.
      </p>
    );
  }

  return (
    <div className="mt-5">
      <h4 className="text-label text-ink-muted mb-2">Ausgeglichene Posten</h4>
      <table className="w-full text-body">
        <thead>
          <tr className="[&>th]:h-7 [&>th]:text-label [&>th]:font-medium [&>th]:text-ink-subtle [&>th]:border-b [&>th]:border-line-strong">
            <th className="text-left w-28">Buchung</th>
            <th className="text-left w-32">Beleg</th>
            <th className="text-left">Partner</th>
            <th className="text-right w-36">Ausgeglichen</th>
            <th className="text-right w-36">Zahlbetrag</th>
            <th className="text-right w-32">Differenz</th>
          </tr>
        </thead>
        <tbody>
          {allocations.map((allocation) => (
            <tr
              key={allocation.id}
              className="[&>td]:h-8 [&>td]:border-b [&>td]:border-line"
            >
              <td className="code-num text-caption text-ink-muted">
                {allocation.openItemEntryNumber}
              </td>
              <td className="code-num text-caption text-ink-muted">
                {allocation.documentNumber || '—'}
              </td>
              <td className="text-ink">{allocation.contactName || '—'}</td>
              <td className="text-right num">{formatCents(allocation.settledAmount, currency)}</td>
              <td className="text-right num">{formatCents(allocation.cashAmount, currency)}</td>
              <td className="text-right num text-ink-muted">
                {allocation.differenceAmount === 0
                  ? '—'
                  : formatCents(allocation.differenceAmount, currency)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// -------------------------------------------------------------------------

/**
 * Erfassung eines Buchungssatzes mit beliebig vielen Zeilen.
 *
 * Die Differenz zwischen Soll und Haben läuft mit. Das ist eine Rechenhilfe und
 * kein Fehler, solange die Buchung nicht abgeschickt ist (§8.3). Der Knopf
 * bleibt deshalb aktiv und nennt beim Klick den Grund.
 */
const BookingForm: React.FC<{
  open: boolean;
  accounts: Account[];
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}> = ({ open, accounts, onOpenChange, onSaved }) => {
  const writeLock = useWriteLock();
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

  const options = useMemo(
    () =>
      accounts
        .filter((a) => !a.isRange && !a.isReserved && a.kontenklasse !== 8)
        .map((a) => ({ value: a.number, label: `${a.number} ${a.name}`, meta: a.kontenklasseName })),
    [accounts],
  );

  function updateLine(index: number, patch: Partial<DraftLine>) {
    setLines((prev) => prev.map((line, i) => (i === index ? { ...line, ...patch } : line)));
  }

  async function submit() {
    if (!balanced) {
      setError(
        `Die Buchung ist nicht ausgeglichen. Soll (${formatCents(debitTotal)}) und Haben (${formatCents(creditTotal)}) müssen übereinstimmen.`,
      );
      return;
    }
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
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Neue Buchung"
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-4 gap-4">
        <Field label="Buchungsdatum" hint="Bestimmt die Periode">
          <Input type="date" value={bookingDate} onChange={(e) => setBookingDate(e.target.value)} />
        </Field>
        <Field label="Belegdatum" hint="Rechnungsdatum">
          <Input
            type="date"
            value={documentDate}
            onChange={(e) => setDocumentDate(e.target.value)}
          />
        </Field>
        <Field label="Leistung von" help="Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG.">
          <Input type="date" value={serviceFrom} onChange={(e) => setServiceFrom(e.target.value)} />
        </Field>
        <Field label="Leistung bis">
          <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
        </Field>
      </div>

      <div className="grid grid-cols-3 gap-4 mt-4">
        <Field label="Buchungstext" className="col-span-2">
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Wofür wurde gebucht?"
          />
        </Field>
        <Field label="Belegnummer" optional>
          <Input
            value={documentNumber}
            onChange={(e) => setDocumentNumber(e.target.value)}
            placeholder="ER-2026-0042"
          />
        </Field>
      </div>

      <div className="mt-8 pt-6 border-t border-line">
        <div className="flex items-center justify-between gap-4 mb-3">
          <span className="flex items-center text-label text-ink-muted">
            Buchungszeilen
            <HelpPopover label="Erklärung zu Buchungszeilen">
              Ein Beleg mit Vorsteuer hat drei Zeilen: Aufwand und Vorsteuer im Soll, die
              Verbindlichkeit im Haben. Bei Reverse Charge sind es vier. Die Summe der Sollzeilen
              muss der Summe der Habenzeilen entsprechen.
            </HelpPopover>
          </span>
          <Button
            variant="quiet"
            size="sm"
            onClick={() => setLines((prev) => [...prev, { side: 'S', account: '', amount: '' }])}
          >
            Zeile hinzufügen
          </Button>
        </div>

        <div className="grid grid-cols-[7rem_1fr_10rem_2rem] gap-x-3 gap-y-2 items-center">
          <span className="text-caption text-ink-subtle">Seite</span>
          <span className="text-caption text-ink-subtle">Konto</span>
          <span className="text-caption text-ink-subtle text-right">Betrag</span>
          <span />

          {lines.map((line, index) => (
            <React.Fragment key={index}>
              <Select
                items={[
                  { value: 'S', label: 'Soll' },
                  { value: 'H', label: 'Haben' },
                ]}
                value={line.side}
                onValueChange={(side) => updateLine(index, { side: side as Side })}
              />
              <Combobox
                items={options}
                value={line.account || null}
                onValueChange={(account) => updateLine(index, { account: account ?? '' })}
                placeholder="Konto suchen …"
                emptyText="Kein Konto gefunden."
              />
              <Input
                align="right"
                inputMode="decimal"
                value={line.amount}
                onChange={(e) => updateLine(index, { amount: e.target.value })}
                placeholder="0,00"
                aria-label={`Betrag Zeile ${index + 1}`}
              />
              {lines.length > 2 ? (
                <Button
                  variant="quiet"
                  size="sm"
                  iconOnly
                  onClick={() => setLines((prev) => prev.filter((_, i) => i !== index))}
                  title="Zeile entfernen"
                  aria-label={`Zeile ${index + 1} entfernen`}
                >
                  <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                </Button>
              ) : (
                <span />
              )}
            </React.Fragment>
          ))}
        </div>

        <div className="mt-4 pt-3 rule-total flex items-center justify-between gap-4 text-body">
          <span className={cn('font-semibold', balanced ? 'text-positive-text' : 'text-ink-muted')}>
            {balanced ? 'Soll und Haben stimmen überein' : 'Noch nicht ausgeglichen'}
          </span>
          <span className="num text-ink">
            Soll {formatCents(debitTotal)} · Haben {formatCents(creditTotal)}
            {!balanced && (
              <span className="text-ink-subtle"> · Differenz {formatCents(debitTotal - creditTotal)}</span>
            )}
          </span>
        </div>
      </div>

      {error && (
        <div className="mt-4 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
          <p className="text-body text-negative-text">{error}</p>
        </div>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const ReverseDialog: React.FC<{
  entry: JournalEntry | null;
  onClose: () => void;
  onDone: () => void;
}> = ({ entry, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (entry) {
      setReason('');
      setError(null);
    }
  }, [entry]);

  async function submit() {
    if (!reason.trim()) {
      setError('Ohne Grund lässt sich die Stornierung später nicht nachvollziehen.');
      return;
    }
    setError(null);
    setBusy(true);
    try {
      await Api.reverseJournalEntry(entry!.id, reason);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={entry !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Buchung stornieren"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="danger"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Stornieren
          </Button>
        </>
      }
    >
      {entry && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{entry.entryNumber}</span> über{' '}
            <span className="num text-ink">{formatCents(grossOf(entry), entry.currency)}</span> wird
            per Generalumkehr zurückgebucht.
            <HelpPopover label="Erklärung zur Generalumkehr">
              Storniert wird mit denselben Konten auf denselben Seiten und negiertem Betrag. Die
              Umsätze der betroffenen Konten gehen dadurch auf null zurück, statt sich wie bei einer
              spiegelverkehrten Gegenbuchung zu verdoppeln. Die Stornobuchung wird auf heute datiert,
              die ursprüngliche Buchung bleibt im Journal sichtbar.
            </HelpPopover>
          </p>

          <Field label="Grund der Stornierung" className="mt-4" error={error ?? undefined}>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Beleg doppelt erfasst"
            />
          </Field>
        </>
      )}
    </Dialog>
  );
};
