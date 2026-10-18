import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Upload } from 'lucide-react';
import type {
  BankAccount,
  BankSuggestion,
  BankSuggestions,
  BankTransaction,
  JournalEntry,
} from '../../types';
import { useNavigation, useRouteObject } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import { usePostingLock } from '../../components/WriteLock';
import { ActorMark } from '../../components/ActorMark';
import { formatCents, formatDate } from '../../utils/formatters';
import {
  Button,
  DetailList,
  DetailPanel,
  EmptyState,
  Notice,
  RadioGroup,
  SearchInput,
  Select,
  RowButton,
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
} from '../../components/ui';
import { PanelHeading, errorText } from './shared';

type Filter = 'open' | 'all';

/** Was vom Umsatz noch keinem Posten zugeordnet ist. */
const remainingOf = (tx: BankTransaction) => Math.max(0, Math.abs(tx.amount) - (tx.matchedAmount || 0));

function matches(tx: BankTransaction, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return [tx.counterpartyName, tx.remittanceInfo, tx.counterpartyIban]
    .filter(Boolean)
    .some((text) => text.toLowerCase().includes(q));
}

/**
 * Was ein Vorschlag beim Bestätigen bucht, oder warum er sich so nicht
 * bestätigen lässt. Ein Vorschlag aus offenen Posten muss den Umsatz genau
 * decken; eine Anzahlung gilt nur mit dem vollen Betrag als vereinnahmt.
 */
function plan(
  tx: BankTransaction,
  suggestion: BankSuggestion,
):
  | { kind: 'direct'; counterAccount: string }
  | { kind: 'advance'; advanceId: number }
  | { kind: 'settle'; allocations: { openItemEntryId: number; settledAmount: number }[] }
  | { kind: 'blocked'; reason: string } {
  if (suggestion.kind === 'rule' || suggestion.kind === 'credit_card') {
    return suggestion.counterAccount
      ? { kind: 'direct', counterAccount: suggestion.counterAccount }
      : { kind: 'blocked', reason: 'Dem Vorschlag fehlt das Gegenkonto.' };
  }
  const items = suggestion.items ?? [];
  const advance = items.find((item) => item.source === 'advance');
  if (advance) {
    return Math.abs(tx.amount) === Math.abs(advance.openAmount)
      ? { kind: 'advance', advanceId: advance.advanceInvoiceId ?? 0 }
      : { kind: 'blocked', reason: 'Eine Anzahlung gilt erst mit dem vollen Betrag als vereinnahmt.' };
  }
  let remaining = remainingOf(tx);
  const allocations = items.map((item) => {
    const amount = Math.min(Math.abs(item.openAmount), remaining);
    remaining -= amount;
    return { openItemEntryId: item.entryId, settledAmount: amount };
  });
  if (remaining !== 0 || allocations.length === 0) {
    return {
      kind: 'blocked',
      reason: `Die Posten decken ${formatCents(remainingOf(tx) - remaining, tx.currency)} von ${formatCents(
        remainingOf(tx),
        tx.currency,
      )}. Den Rest ordnet Claude zu.`,
    };
  }
  return { kind: 'settle', allocations: allocations.filter((a) => a.settledAmount > 0) };
}

/**
 * Die Bankumsätze: offene zuerst. Zu einem offenen Umsatz zeigt die
 * Seitenleiste die Vorschläge des Backends, und der Mensch bestätigt einen
 * davon. Freie Zuordnungen und Splitbuchungen erledigt Claude.
 */
export const BankTab: React.FC = () => {
  const { year } = useWorkspace();
  const { id, open, close } = useRouteObject('books', 'bank');
  const lock = usePostingLock();
  const [transactions, setTransactions] = useState<BankTransaction[]>([]);
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>('open');
  const [search, setSearch] = useState('');
  const [importing, setImporting] = useState(false);
  const [importNotice, setImportNotice] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [txs, journal, banks] = await Promise.all([
        Api.getBankTransactions(),
        Api.getJournalEntries(),
        Api.getBankAccounts(),
      ]);
      setTransactions([...txs].sort((a, b) => b.bookingDate.localeCompare(a.bookingDate) || b.id - a.id));
      setEntries(journal ?? []);
      setAccounts(banks);
      setError(null);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    void load();
  }, [load, year]);

  const entryOf = useMemo(() => {
    const map = new Map<number, JournalEntry>();
    for (const entry of entries) {
      if (entry.bankTxId && entry.kind !== 'reversal' && !map.has(entry.bankTxId))
        map.set(entry.bankTxId, entry);
    }
    return map;
  }, [entries]);

  const openCount = transactions.filter((t) => t.matchStatus === 'unmatched').length;
  const visible = transactions
    .filter((t) => filter === 'all' || t.matchStatus === 'unmatched')
    .filter((t) => matches(t, search.trim()));
  const selected = id ? transactions.find((t) => String(t.id) === id) : undefined;
  const accountName = (iban: string) => accounts.find((a) => a.iban === iban)?.name;

  /**
   * Liest einen Kontoauszug über seinen Pfad ein; die Datei wird dabei als
   * Beleg abgelegt. Ein unbekanntes Konto legen Sie zuerst in den Stammdaten an.
   */
  async function importStatement() {
    setImporting(true);
    setImportNotice(null);
    try {
      const path = await Api.selectStatementFile();
      if (!path) return;
      const preview = await Api.previewBankStatement(path);
      const unknown = preview.accounts.filter((a) => !accounts.some((known) => known.iban === a.iban));
      if (unknown.length > 0) {
        setImportNotice(
          `Das Konto ${unknown.map((a) => a.iban).join(', ')} ist noch nicht angelegt. Legen Sie es unter Unternehmen → Stammdaten → Bankkonten an.`,
        );
        return;
      }
      const count = await Api.importCAMTFile(path, '');
      toast.success(`${count} ${count === 1 ? 'Umsatz' : 'Umsätze'} eingelesen.`);
      await load();
    } catch (e) {
      toast.error(errorText(e));
    } finally {
      setImporting(false);
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Gegenüber oder Verwendungszweck"
          aria-label="Bankumsätze durchsuchen"
          className="max-w-sm"
        />
        <Select<Filter>
          className="w-52"
          aria-label="Bankumsätze filtern"
          items={[
            { value: 'open', label: `Offen (${openCount})` },
            { value: 'all', label: `Alle Umsätze (${transactions.length})` },
          ]}
          value={filter}
          onValueChange={setFilter}
        />
        <div className="ml-auto">
          <Button
            variant="secondary"
            icon={<Upload className="w-4 h-4" strokeWidth={1.5} />}
            loading={importing}
            disabled={lock.locked}
            title={lock.hint ?? 'Kontoauszug im Format CAMT.053 einlesen'}
            onClick={() => void importStatement()}
          >
            Kontoauszug einlesen
          </Button>
        </div>
      </div>

      {importNotice && <Notice className="mt-4" text={importNotice} />}

      <div className="mt-4">
        {error ? (
          <Notice tone="negative" text={error} />
        ) : loading ? (
          <SkeletonRows rows={8} />
        ) : transactions.length === 0 ? (
          <EmptyState title="Noch keine Bankumsätze" description="Claude liest Kontoauszüge für Sie ein." />
        ) : visible.length === 0 ? (
          <EmptyState
            variant={filter === 'open' && !search ? 'leer' : 'gefiltert'}
            title={
              filter === 'open' && !search ? 'Alle Umsätze sind zugeordnet' : 'Kein Umsatz passt zur Suche'
            }
            action={
              <Button
                variant="secondary"
                onClick={() => {
                  setFilter('all');
                  setSearch('');
                }}
              >
                Alle Umsätze zeigen
              </Button>
            }
          />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-28">Datum</Th>
                <Th>Gegenüber und Verwendungszweck</Th>
                <Th numeric className="w-32">
                  Betrag
                </Th>
                <Th className="w-10">
                  <span className="sr-only">Gebucht von</span>
                </Th>
                <Th className="w-32">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {visible.map((tx) => {
                const entry = entryOf.get(tx.id);
                return (
                  <Tr
                    key={tx.id}
                    variant={String(tx.id) === id ? 'selected' : 'default'}
                    className="cursor-pointer"
                    onClick={() => open(String(tx.id))}
                  >
                    <Td className="text-ink-subtle num">
                      <RowButton
                        aria-label={`Umsatz vom ${formatDate(tx.bookingDate)}, ${tx.counterpartyName || 'ohne Gegenüber'}, ${formatCents(tx.amount, tx.currency)}`}
                        onOpen={() => open(String(tx.id))}
                      >
                        {formatDate(tx.bookingDate)}
                      </RowButton>
                    </Td>
                    <Td fill>
                      {tx.counterpartyName || '—'}
                      <span className="text-ink-subtle"> · {tx.remittanceInfo}</span>
                    </Td>
                    <Td numeric className={cn(tx.amount < 0 && 'text-ink-muted')}>
                      {formatCents(tx.amount, tx.currency)}
                    </Td>
                    <Td className="px-2">
                      {entry && <ActorMark actor={entry.actor} channel={entry.channel} />}
                    </Td>
                    <Td>
                      <TxState tx={tx} />
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </div>

      <BankPanel
        tx={selected}
        entry={selected ? entryOf.get(selected.id) : undefined}
        accountName={selected ? accountName(selected.accountIban) : undefined}
        onClose={close}
        onBooked={async () => {
          await load();
          close();
        }}
      />
    </div>
  );
};

const TxState: React.FC<{ tx: BankTransaction }> = ({ tx }) => {
  if (tx.matchStatus === 'matched') return <StatusBadge status="zugeordnet" />;
  if (tx.matchStatus === 'ignored') return <span className="text-caption text-ink-subtle">Ohne Buchung</span>;
  return <StatusBadge status={tx.matchedAmount > 0 ? 'teilweiseAusgeglichen' : 'offen'} />;
};

/** Die Seitenleiste eines Bankumsatzes; bei einem offenen mit den Vorschlägen zum Bestätigen. */
const BankPanel: React.FC<{
  tx: BankTransaction | undefined;
  entry: JournalEntry | undefined;
  accountName?: string;
  onClose: () => void;
  onBooked: () => Promise<void>;
}> = ({ tx, entry, accountName, onClose, onBooked }) => {
  const { navigate } = useNavigation();
  const lock = usePostingLock();
  const [shown, setShown] = useState<BankTransaction | undefined>(tx);
  const [suggestions, setSuggestions] = useState<BankSuggestions | null>(null);
  const [choice, setChoice] = useState(0);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);

  useEffect(() => {
    if (tx) setShown(tx);
  }, [tx]);

  useEffect(() => {
    setSuggestions(null);
    setChoice(0);
    setFailure(null);
    if (!tx || tx.matchStatus !== 'unmatched') return;
    let cancelled = false;
    Api.suggestBankMatches(tx.id)
      .then((result) => !cancelled && setSuggestions(result))
      .catch(() => !cancelled && setSuggestions({ bankTxId: tx.id, amount: tx.amount, suggestions: [] }));
    return () => {
      cancelled = true;
    };
  }, [tx]);

  if (!shown) return null;

  const options = suggestions?.suggestions ?? [];
  const chosen = options[choice];
  const action = chosen ? plan(shown, chosen) : null;

  async function confirm() {
    if (!shown || !action) return;
    setBusy(true);
    setFailure(null);
    try {
      if (action.kind === 'direct') {
        await Api.bookBankTransactionDirect(
          shown.id,
          action.counterAccount,
          `${shown.counterpartyName} – ${shown.remittanceInfo}`,
        );
      } else if (action.kind === 'advance') {
        await Api.settleAdvance({
          advanceId: action.advanceId,
          bankTxId: shown.id,
          paymentDate: shown.bookingDate,
          paymentAccount: shown.ledgerAccount,
        });
      } else if (action.kind === 'settle') {
        await Api.settlePayment({
          bankTxId: shown.id,
          paymentAccount: shown.ledgerAccount,
          paymentDate: shown.bookingDate,
          valueDate: shown.valueDate,
          allocations: action.allocations.map((a) => ({
            ...a,
            differenceKind: 'none' as const,
            differenceAmount: 0,
          })),
        });
      } else {
        return;
      }
      toast.success('Bankumsatz gebucht.');
      await onBooked();
    } catch (e) {
      setFailure(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const facts = [
    { label: 'Betrag', value: formatCents(shown.amount, shown.currency), numeric: true },
    { label: 'Buchungstag', value: formatDate(shown.bookingDate), numeric: true },
    ...(shown.valueDate && shown.valueDate !== shown.bookingDate
      ? [{ label: 'Wertstellung', value: formatDate(shown.valueDate), numeric: true }]
      : []),
    { label: 'Gegenüber', value: shown.counterpartyName || '—' },
    ...(shown.counterpartyIban
      ? [{ label: 'IBAN', value: <span className="code-num">{shown.counterpartyIban}</span> }]
      : []),
    { label: 'Verwendungszweck', value: shown.remittanceInfo || '—' },
    { label: 'Konto', value: `${shown.ledgerAccount}${accountName ? ` · ${accountName}` : ''}` },
  ];

  const unmatched = shown.matchStatus === 'unmatched';

  return (
    <DetailPanel
      open={Boolean(tx)}
      onOpenChange={(next) => !next && onClose()}
      title={shown.counterpartyName || 'Bankumsatz'}
      context={`${formatDate(shown.bookingDate)} · ${formatCents(shown.amount, shown.currency)}`}
      meta={<TxState tx={shown} />}
      footer={
        unmatched && chosen ? (
          <Button
            variant="primary"
            loading={busy}
            disabled={lock.locked || action?.kind === 'blocked'}
            title={lock.hint ?? (action?.kind === 'blocked' ? action.reason : undefined)}
            onClick={() => void confirm()}
          >
            Vorschlag bestätigen
          </Button>
        ) : undefined
      }
    >
      <DetailList items={facts} />

      {entry && (
        <div className="mt-6">
          <PanelHeading>Buchung</PanelHeading>
          <div className="flex items-center gap-3 text-body">
            <button
              type="button"
              className="code-num text-caption text-accent-text hover:underline"
              onClick={() => navigate({ area: 'books', tab: 'journal', id: entry.entryNumber })}
            >
              {entry.entryNumber}
            </button>
            <span className="flex-1 truncate">{entry.description}</span>
            <ActorMark actor={entry.actor} channel={entry.channel} />
          </div>
        </div>
      )}

      {unmatched && (
        <div className="mt-6">
          <PanelHeading>Vorschläge</PanelHeading>
          {!suggestions ? (
            <SkeletonRows rows={3} />
          ) : options.length === 0 ? (
            <p className="text-body text-ink-muted">
              {suggestions.note || 'Buchfink hat keinen passenden Posten gefunden.'} Claude ordnet den Umsatz
              zu, wenn Sie darum bitten.
            </p>
          ) : (
            <RadioGroup<number>
              value={choice}
              onValueChange={setChoice}
              options={options.map((suggestion, index) => ({
                value: index,
                label: (
                  <span className="flex items-baseline gap-3">
                    <span className="flex-1">{suggestion.label}</span>
                    <span className="num text-ink-muted">
                      {formatCents(suggestion.amount, shown.currency)}
                    </span>
                  </span>
                ),
                hint: suggestion.reasons.join(' · '),
              }))}
            />
          )}
          {action?.kind === 'blocked' ? (
            <p className="mt-3 text-caption text-attention-text">{action.reason}</p>
          ) : (
            options.length > 0 && (
              <p className="mt-3 text-caption text-ink-subtle">
                Bestätigen bucht den Umsatz gegen den gewählten Vorschlag.
              </p>
            )
          )}
          {failure && <Notice tone="negative" className="mt-4" text={failure} />}
        </div>
      )}
    </DetailPanel>
  );
};
