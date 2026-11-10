import React, { useCallback, useEffect, useMemo, useState } from 'react';
import type { Account, JournalEntry, PaymentAllocationDetail } from '../../types';
import { useNavigation, useRouteObject } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { useLatestRequest, useLiveReload } from '../../live';
import { Api } from '../../services/api';
import { ActorMark } from '../../components/ActorMark';
import { actorWho } from '../../utils/actor';
import { JournalFilterView } from '../../components/JournalFilterView';
import {
  formatCents,
  formatDate,
  formatDateRange,
  formatDateTime,
  formatShortHash,
} from '../../utils/formatters';
import {
  Button,
  DetailList,
  DetailPanel,
  Disclosure,
  EmptyState,
  Notice,
  SearchInput,
  RowButton,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from '../../components/ui';
import { PanelHeading, PostingLines, entryStatus, errorText, grossOf, reversedIds } from './shared';

const SOURCE_LABELS: Record<string, string> = {
  manual: 'Handbuchung',
  receipt: 'Eingangsbeleg',
  invoice: 'Ausgangsrechnung',
  payment: 'Zahlung',
  opening: 'Eröffnungsbilanz',
  depreciation: 'Abschreibung',
  closing: 'Abschlussbuchung',
};

/** So viele Buchungen zeigt die Liste, bevor sie nachlädt. */
const PAGE = 100;

/** Konten in der Schreibweise Soll → Haben. */
function accountPath(entry: JournalEntry): string {
  const debit = entry.lines.filter((l) => l.side === 'S').map((l) => l.account);
  const credit = entry.lines.filter((l) => l.side === 'H').map((l) => l.account);
  return `${debit.join(' · ')} → ${credit.join(' · ')}`;
}

/** Die Generalumkehr nennt die Buchung, die sie zurücknimmt (§11.2). */
function stornoReference(entry: JournalEntry, numberOf: Map<number, string>): string | undefined {
  const origin = entry.reversalOfId ? numberOf.get(entry.reversalOfId) : undefined;
  return entry.kind === 'reversal' && origin ? `Storno zu ${origin}` : undefined;
}

function matches(entry: JournalEntry, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return (
    entry.entryNumber.toLowerCase().includes(q) ||
    entry.description.toLowerCase().includes(q) ||
    (entry.documentNumber ?? '').toLowerCase().includes(q) ||
    entry.lines.some((line) => line.account.startsWith(q))
  );
}

/**
 * Das Journal: alle Buchungen des Geschäftsjahres, die neueste zuerst. Die
 * Liste zeigt Text, Konten, Betrag und Zustand; Buchungssatz, Beleg, Bearbeiter
 * und Festschreibung stehen in der Seitenleiste. Gebucht und storniert wird
 * über Claude, in der App gibt es keine freie Buchung.
 */
export const JournalTab: React.FC = () => {
  const { navigate } = useNavigation();
  const { year } = useWorkspace();
  const { id, open, close } = useRouteObject('books', 'journal');
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [limit, setLimit] = useState(PAGE);

  const begin = useLatestRequest();
  const load = useCallback(async () => {
    const latest = begin();
    try {
      const list = await Api.getJournalEntries();
      if (!latest()) return;
      setEntries([...(list ?? [])].sort((a, b) => b.id - a.id));
      setError(null);
    } catch (e) {
      if (latest()) setError(errorText(e));
    } finally {
      if (latest()) setLoading(false);
    }
  }, [begin]);

  useEffect(() => {
    setLoading(true);
    void load();
  }, [load, year]);

  useLiveReload(load, ['buchungen', 'belege', 'bank', 'abschluss']);

  const reversed = useMemo(() => reversedIds(entries), [entries]);
  const visible = useMemo(() => entries.filter((e) => matches(e, search.trim())), [entries, search]);
  const selected = id ? entries.find((entry) => entry.entryNumber === id) : undefined;
  const numberOf = useMemo(() => new Map(entries.map((e) => [e.id, e.entryNumber])), [entries]);

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setLimit(PAGE);
          }}
          placeholder="Buchung, Text, Beleg oder Konto"
          aria-label="Buchungen durchsuchen"
          className="max-w-md"
        />
        <p className="text-caption text-ink-subtle num">
          {loading ? '' : `${visible.length} von ${entries.length} Buchungen`}
        </p>
      </div>

      <div className="mt-4">
        {error ? (
          <Notice tone="negative" text={error} />
        ) : loading ? (
          <SkeletonRows rows={8} />
        ) : entries.length === 0 ? (
          <EmptyState
            title="Noch keine Buchungen"
            description={`Im Geschäftsjahr ${year} ist noch nichts gebucht.`}
          />
        ) : visible.length === 0 ? (
          <EmptyState
            variant="gefiltert"
            title="Keine Buchung passt zur Suche"
            action={
              <Button variant="secondary" onClick={() => setSearch('')}>
                Suche zurücksetzen
              </Button>
            }
          />
        ) : (
          <>
            <Table density="kompakt">
              <Thead sticky>
                <Tr>
                  <Th className="w-32">Buchung</Th>
                  <Th className="w-28">Datum</Th>
                  <Th>Buchungstext</Th>
                  <Th optional className="w-48">
                    Soll → Haben
                  </Th>
                  <Th numeric className="w-32">
                    Betrag
                  </Th>
                  <Th className="w-10">
                    <span className="sr-only">Erfasst von</span>
                  </Th>
                  <Th className="w-36">Stand</Th>
                </Tr>
              </Thead>
              <Tbody>
                {visible.slice(0, limit).map((entry) => {
                  const status = entryStatus(entry, reversed);
                  return (
                    <Tr
                      key={entry.id}
                      variant={
                        status === 'storniert' ? 'storno' : entry.entryNumber === id ? 'selected' : 'default'
                      }
                      className="cursor-pointer"
                      onClick={() => open(entry.entryNumber)}
                    >
                      <Td code>
                        <RowButton onOpen={() => open(entry.entryNumber)}>{entry.entryNumber}</RowButton>
                      </Td>
                      <Td className="text-ink-subtle num">{formatDate(entry.bookingDate)}</Td>
                      <Td fill title={entry.description}>
                        {entry.description}
                      </Td>
                      <Td code optional className="max-w-[12rem] truncate">
                        {accountPath(entry)}
                      </Td>
                      <Td numeric>{formatCents(grossOf(entry), entry.currency)}</Td>
                      <Td className="px-2">
                        <ActorMark actor={entry.actor} channel={entry.channel} />
                      </Td>
                      <Td>
                        <StatusBadge status={status} reference={stornoReference(entry, numberOf)} />
                      </Td>
                    </Tr>
                  );
                })}
              </Tbody>
            </Table>
            {visible.length > limit && (
              <div className="mt-4 flex justify-center">
                <Button variant="secondary" onClick={() => setLimit((n) => n + PAGE)}>
                  {`Weitere ${Math.min(PAGE, visible.length - limit)} Buchungen anzeigen`}
                </Button>
              </div>
            )}
          </>
        )}
      </div>

      <Disclosure title="Auswerten und exportieren" context="Zeilen filtern und als CSV speichern">
        <JournalAnalysis year={year} />
      </Disclosure>

      <EntryPanel
        entry={selected}
        entries={entries}
        reversed={reversed}
        onClose={close}
        onOpenEntry={open}
        onOpenReceipt={(receiptId) => navigate({ area: 'books', tab: 'receipts', id: String(receiptId) })}
      />
    </div>
  );
};

/** Die Zeilenauswertung lädt den Kontenrahmen erst, wenn sie aufgeklappt wird. */
const JournalAnalysis: React.FC<{ year: number }> = ({ year }) => {
  const { navigate } = useNavigation();
  const [accounts, setAccounts] = useState<Account[] | null>(null);
  useEffect(() => {
    Api.getAccounts()
      .then(setAccounts)
      .catch(() => setAccounts([]));
  }, []);
  if (!accounts) return <SkeletonRows rows={4} />;
  return <JournalFilterView accounts={accounts} year={year} onNavigate={navigate} />;
};

/** Die Seitenleiste einer Buchung. */
const EntryPanel: React.FC<{
  entry: JournalEntry | undefined;
  entries: JournalEntry[];
  reversed: Set<number>;
  onClose: () => void;
  onOpenEntry: (entryNumber: string) => void;
  onOpenReceipt: (receiptId: number) => void;
}> = ({ entry, entries, reversed, onClose, onOpenEntry, onOpenReceipt }) => {
  // Der Inhalt bleibt beim Schließen stehen, bis die Seitenleiste ausgeblendet ist.
  const [shown, setShown] = useState<JournalEntry | undefined>(entry);
  useEffect(() => {
    if (entry) setShown(entry);
  }, [entry]);
  if (!shown) return null;

  const byId = (entryId?: number) => entries.find((e) => e.id === entryId);
  const origin = byId(shown.reversalOfId);
  const reversal = entries.find((e) => e.reversalOfId === shown.id);
  const corrects = byId(shown.correctsEntryId);
  const correctedBy = entries.find((e) => e.correctsEntryId === shown.id);
  const status = entryStatus(shown, reversed);

  const entryLink = (target: JournalEntry) => (
    <button
      type="button"
      className="code-num text-accent-text hover:underline"
      onClick={() => onOpenEntry(target.entryNumber)}
    >
      {target.entryNumber}
    </button>
  );

  const facts = [
    { label: 'Buchungsdatum', value: formatDate(shown.bookingDate), numeric: true },
    {
      label: 'Beleg',
      value: `${formatDate(shown.documentDate)}${shown.documentNumber ? ` · ${shown.documentNumber}` : ''}`,
      numeric: true,
    },
    { label: 'Leistung', value: formatDateRange(shown.serviceDateFrom, shown.serviceDateTo), numeric: true },
    { label: 'Herkunft', value: SOURCE_LABELS[shown.source] ?? shown.source },
    {
      label: 'Erfasst von',
      value: (
        <span className="inline-flex items-center gap-1.5">
          <ActorMark actor={shown.actor} channel={shown.channel} className="-ml-1" />
          {actorWho(shown.actor, shown.channel)}
        </span>
      ),
    },
    {
      label: 'Festgeschrieben',
      value: shown.committedAt ? formatDateTime(shown.committedAt) : 'noch nicht',
    },
    ...(origin ? [{ label: 'Storno zu', value: entryLink(origin) }] : []),
    ...(reversal ? [{ label: 'Storniert durch', value: entryLink(reversal) }] : []),
    ...(shown.reversalReason ? [{ label: 'Grund', value: shown.reversalReason }] : []),
    ...(corrects ? [{ label: 'Berichtigt', value: entryLink(corrects) }] : []),
    ...(correctedBy ? [{ label: 'Berichtigt durch', value: entryLink(correctedBy) }] : []),
  ];

  return (
    <DetailPanel
      open={Boolean(entry)}
      onOpenChange={(next) => !next && onClose()}
      title={shown.entryNumber}
      context={shown.description}
      meta={
        <StatusBadge status={status} reference={origin ? `Storno zu ${origin.entryNumber}` : undefined} />
      }
      width="lg"
      footer={
        shown.receiptId ? (
          <Button
            variant="secondary"
            onClick={() => onOpenReceipt(shown.receiptId as number)}
            aria-label={`Beleg zur Buchung ${shown.entryNumber} anzeigen`}
          >
            Beleg anzeigen
          </Button>
        ) : undefined
      }
    >
      <PostingLines lines={shown.lines} currency={shown.currency} />

      <DetailList className="mt-6" items={facts} />

      {status === 'gebucht' && (
        <p className="mt-5 text-caption text-ink-subtle">
          Bis zur Festschreibung lässt sich die Buchung per Storno korrigieren. Bitten Sie Claude darum.
        </p>
      )}

      {shown.source === 'payment' && <PaymentAllocations entryId={shown.id} currency={shown.currency} />}

      <Disclosure title="Prüfdetails" context="Prüfsummen der Buchungskette">
        <DetailList
          items={[
            {
              label: 'Prüfsumme',
              value: <span className="font-mono text-caption">{formatShortHash(shown.entryHash)}</span>,
            },
            {
              label: 'Vorgängerin',
              value: <span className="font-mono text-caption">{formatShortHash(shown.previousHash)}</span>,
            },
          ]}
        />
      </Disclosure>
    </DetailPanel>
  );
};

/** Wogegen eine Zahlung lief: die ausgeglichenen Posten einer Zahlungsbuchung. */
const PaymentAllocations: React.FC<{ entryId: number; currency: string }> = ({ entryId, currency }) => {
  const [rows, setRows] = useState<PaymentAllocationDetail[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Api.getPaymentAllocations(entryId)
      .then((list) => !cancelled && setRows(list))
      .catch((e) => !cancelled && setError(errorText(e)));
    return () => {
      cancelled = true;
    };
  }, [entryId]);

  return (
    <div className="mt-6">
      <PanelHeading>Ausgeglichene Posten</PanelHeading>
      {error ? (
        <p className="text-caption text-negative-text">{error}</p>
      ) : !rows ? (
        <SkeletonRows rows={2} />
      ) : rows.length === 0 ? (
        <p className="text-caption text-ink-subtle">Diese Zahlung ist keinem offenen Posten zugeordnet.</p>
      ) : (
        <ul className="divide-y divide-line border-y border-line">
          {rows.map((row) => (
            <li key={row.id} className="flex items-center gap-3 py-2 text-body">
              <span className="code-num text-caption text-ink-muted w-28 shrink-0">
                {row.documentNumber || row.openItemEntryNumber}
              </span>
              <span className="flex-1 truncate">{row.contactName || '—'}</span>
              <span className="num">{formatCents(row.settledAmount, currency)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
