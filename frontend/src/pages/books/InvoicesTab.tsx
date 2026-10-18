import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FilePlus } from 'lucide-react';
import type { Invoice, JournalEntry } from '../../types';
import { useNavigation, useRouteObject } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import { usePostingLock } from '../../components/WriteLock';
import { ActorMark } from '../../components/ActorMark';
import { formatCents, formatDate, formatDateRange, formatQuantity } from '../../utils/formatters';
import {
  Button,
  DetailList,
  DetailPanel,
  EmptyState,
  Notice,
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
  toast,
  type Status,
} from '../../components/ui';
import { DocumentPreview, PanelHeading, errorText, todayISO } from './shared';
import { InvoiceDialog } from './InvoiceDialog';

const KIND_LABEL: Record<Invoice['kind'], string> = {
  invoice: 'Rechnung',
  advance: 'Abschlagsrechnung',
  final: 'Schlussrechnung',
  correction: 'Rechnungskorrektur',
  cancellation: 'Stornorechnung',
};

type Filter = 'all' | 'open' | 'overdue';

/** Ältere Rechnungen haben die Dokumentart noch nicht; leer heißt Rechnung. */
const kindOf = (invoice: Invoice): Invoice['kind'] => invoice.kind || 'invoice';

const isOpen = (invoice: Invoice) =>
  (invoice.status === 'issued' || invoice.status === 'issued_pending_document') &&
  kindOf(invoice) !== 'cancellation';

const isOverdue = (invoice: Invoice, today: string) =>
  isOpen(invoice) && Boolean(invoice.dueDate) && invoice.dueDate < today;

/** Das Statuswort einer Zeile; die Stornorechnung nennt die Rechnung, die sie zurücknimmt. */
function statusOf(invoice: Invoice, today: string): { status: Status; reference?: string } {
  if (kindOf(invoice) === 'cancellation') {
    return {
      status: 'gebucht',
      reference: invoice.correctsInvoiceNumber ? `Storno zu ${invoice.correctsInvoiceNumber}` : undefined,
    };
  }
  switch (invoice.status) {
    case 'draft':
      return { status: 'entwurf' };
    case 'paid':
      return { status: 'ausgeglichen' };
    case 'cancelled':
      return { status: 'storniert' };
    default:
      if (isOverdue(invoice, today)) return { status: 'ueberfaellig' };
      return { status: invoice.paidAmount > 0 ? 'teilweiseAusgeglichen' : 'offen' };
  }
}

/**
 * Die Ausgangsrechnungen mit ihrem Zahlungsstand. Das Dokument, die Positionen
 * und die Buchung stehen in der Seitenleiste. Geschrieben wird eine Rechnung
 * im Gespräch mit Claude; ohne Claude über die Notfallmaske.
 */
export const InvoicesTab: React.FC = () => {
  const { year } = useWorkspace();
  const { id, open, close } = useRouteObject('books', 'invoices');
  const lock = usePostingLock();
  const today = todayISO();
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>('all');
  const [search, setSearch] = useState('');
  const [writing, setWriting] = useState(false);

  const load = useCallback(async () => {
    try {
      const [list, journal] = await Promise.all([Api.getInvoices(), Api.getJournalEntries()]);
      setInvoices([...list].sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id));
      setEntries(journal ?? []);
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

  const entryById = useMemo(() => new Map(entries.map((e) => [e.id, e])), [entries]);

  const openCount = invoices.filter(isOpen).length;
  const overdueCount = invoices.filter((i) => isOverdue(i, today)).length;
  const query = search.trim().toLowerCase();
  const visible = invoices
    .filter((i) => (filter === 'open' ? isOpen(i) : filter === 'overdue' ? isOverdue(i, today) : true))
    .filter(
      (i) =>
        !query ||
        i.invoiceNumber.toLowerCase().includes(query) ||
        i.contactName.toLowerCase().includes(query),
    );
  const pendingDocument = invoices.filter((i) => i.status === 'issued_pending_document');
  const selected = id ? invoices.find((i) => String(i.id) === id) : undefined;

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Rechnungsnummer oder Kunde"
          aria-label="Rechnungen durchsuchen"
          className="max-w-sm"
        />
        <Select<Filter>
          className="w-52"
          aria-label="Rechnungen filtern"
          items={[
            { value: 'all', label: `Alle Rechnungen (${invoices.length})` },
            { value: 'open', label: `Offen (${openCount})` },
            { value: 'overdue', label: `Überfällig (${overdueCount})` },
          ]}
          value={filter}
          onValueChange={setFilter}
        />
        <div className="ml-auto">
          <Button
            variant="secondary"
            icon={<FilePlus className="w-4 h-4" strokeWidth={1.5} />}
            disabled={lock.locked}
            title={lock.hint}
            onClick={() => setWriting(true)}
          >
            Rechnung schreiben
          </Button>
        </div>
      </div>

      {pendingDocument.length > 0 && (
        <Notice
          className="mt-4"
          text={`Zu ${pendingDocument.length === 1 ? 'einer Rechnung' : `${pendingDocument.length} Rechnungen`} fehlt das Dokument. Die Nummer bleibt vergeben.`}
        />
      )}

      <div className="mt-4">
        {error ? (
          <Notice tone="negative" text={error} />
        ) : loading ? (
          <SkeletonRows rows={8} />
        ) : invoices.length === 0 ? (
          <EmptyState
            title="Noch keine Rechnungen"
            description={`Im Geschäftsjahr ${year} ist noch keine Rechnung ausgestellt.`}
          />
        ) : visible.length === 0 ? (
          <EmptyState
            variant="gefiltert"
            title="Keine Rechnung passt zum Filter"
            action={
              <Button
                variant="secondary"
                onClick={() => {
                  setFilter('all');
                  setSearch('');
                }}
              >
                Filter zurücksetzen
              </Button>
            }
          />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-36">Rechnung</Th>
                <Th optional className="w-28">
                  Datum
                </Th>
                <Th>Kunde</Th>
                <Th numeric className="w-32">
                  Brutto
                </Th>
                <Th optional className="w-28">
                  Fällig
                </Th>
                <Th className="w-10">
                  <span className="sr-only">Ausgestellt von</span>
                </Th>
                <Th className="w-44">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {visible.map((invoice) => {
                const state = statusOf(invoice, today);
                const entry = invoice.journalEntryId ? entryById.get(invoice.journalEntryId) : undefined;
                return (
                  <Tr
                    key={invoice.id}
                    variant={
                      invoice.status === 'cancelled'
                        ? 'storno'
                        : String(invoice.id) === id
                          ? 'selected'
                          : 'default'
                    }
                    className="cursor-pointer"
                    onClick={() => open(String(invoice.id))}
                  >
                    <Td code>
                      <RowButton onOpen={() => open(String(invoice.id))}>
                        {invoice.invoiceNumber || 'Entwurf'}
                      </RowButton>
                      {kindOf(invoice) !== 'invoice' && (
                        <span className="block text-ink-subtle font-normal">
                          {KIND_LABEL[kindOf(invoice)]}
                        </span>
                      )}
                    </Td>
                    <Td optional className="text-ink-subtle num">
                      {formatDate(invoice.date)}
                    </Td>
                    <Td fill>{invoice.contactName}</Td>
                    <Td numeric>{formatCents(invoice.grossAmount, invoice.currency)}</Td>
                    <Td optional className="text-ink-subtle num">
                      {invoice.dueDate ? formatDate(invoice.dueDate) : '—'}
                    </Td>
                    <Td className="px-2">
                      {entry && <ActorMark actor={entry.actor} channel={entry.channel} />}
                    </Td>
                    <Td>
                      <StatusBadge status={state.status} reference={state.reference} />
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </div>

      <InvoicePanel
        invoice={selected}
        invoices={invoices}
        entry={selected?.journalEntryId ? entryById.get(selected.journalEntryId) : undefined}
        onClose={close}
        onChanged={load}
      />

      {writing && (
        <InvoiceDialog
          onClose={() => setWriting(false)}
          onIssued={async (invoice) => {
            setWriting(false);
            toast.success(`Rechnung ${invoice.invoiceNumber} ausgestellt.`);
            await load();
            open(String(invoice.id));
          }}
        />
      )}
    </div>
  );
};

/** Die Seitenleiste einer Rechnung: Dokument, Angaben, Positionen, Buchung. */
const InvoicePanel: React.FC<{
  invoice: Invoice | undefined;
  invoices: Invoice[];
  entry: JournalEntry | undefined;
  onClose: () => void;
  onChanged: () => Promise<void>;
}> = ({ invoice, invoices, entry, onClose, onChanged }) => {
  const { navigate } = useNavigation();
  const { open } = useRouteObject('books', 'invoices');
  const lock = usePostingLock();
  const today = todayISO();
  const [shown, setShown] = useState<Invoice | undefined>(invoice);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (invoice) setShown(invoice);
  }, [invoice]);
  if (!shown) return null;

  const state = statusOf(shown, today);
  const cancelledBy = invoices.find((i) => i.id === shown.cancelledByInvoiceId);
  const corrects = invoices.find((i) => i.id === shown.correctsInvoiceId);
  const invoiceLink = (target: Invoice) => (
    <button
      type="button"
      className="code-num text-accent-text hover:underline"
      onClick={() => open(String(target.id))}
    >
      {target.invoiceNumber}
    </button>
  );

  async function regenerate() {
    if (!shown) return;
    setBusy(true);
    try {
      await Api.regenerateInvoiceDocument(shown.id);
      toast.success(`Dokument zu ${shown.invoiceNumber} erzeugt.`);
      await onChanged();
    } catch (e) {
      toast.error(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const facts = [
    { label: 'Art', value: KIND_LABEL[kindOf(shown)] },
    { label: 'Rechnungsdatum', value: formatDate(shown.date), numeric: true },
    { label: 'Leistung', value: formatDateRange(shown.serviceDateFrom, shown.serviceDateTo), numeric: true },
    { label: 'Fällig', value: shown.dueDate ? formatDate(shown.dueDate) : '—', numeric: true },
    {
      label: 'Betrag',
      value: `${formatCents(shown.grossAmount, shown.currency)} · netto ${formatCents(shown.netAmount, shown.currency)}`,
      numeric: true,
    },
    ...(shown.paidAmount > 0
      ? [{ label: 'Bezahlt', value: formatCents(shown.paidAmount, shown.currency), numeric: true }]
      : []),
    ...(shown.sentAt ? [{ label: 'Versendet', value: formatDate(shown.sentAt), numeric: true }] : []),
    ...(corrects ? [{ label: 'Bezieht sich auf', value: invoiceLink(corrects) }] : []),
    ...(cancelledBy ? [{ label: 'Storniert durch', value: invoiceLink(cancelledBy) }] : []),
    ...(entry
      ? [
          {
            label: 'Buchung',
            value: (
              <span className="inline-flex items-center gap-2">
                <button
                  type="button"
                  className="code-num text-accent-text hover:underline"
                  onClick={() => navigate({ area: 'books', tab: 'journal', id: entry.entryNumber })}
                >
                  {entry.entryNumber}
                </button>
                <ActorMark actor={entry.actor} channel={entry.channel} />
              </span>
            ),
          },
        ]
      : []),
  ];

  return (
    <DetailPanel
      open={Boolean(invoice)}
      onOpenChange={(next) => !next && onClose()}
      title={shown.invoiceNumber || 'Entwurf'}
      context={shown.contactName}
      meta={<StatusBadge status={state.status} reference={state.reference} />}
      width="lg"
      footer={
        shown.status === 'issued_pending_document' ? (
          <Button
            variant="primary"
            loading={busy}
            disabled={lock.locked}
            title={lock.hint}
            onClick={() => void regenerate()}
          >
            Dokument erzeugen
          </Button>
        ) : undefined
      }
    >
      {shown.receiptId && shown.status !== 'issued_pending_document' && (
        <DocumentPreview
          className="mb-6"
          load={() => Api.getInvoiceDocument(shown.id)}
          loadKey={`invoice-${shown.id}-${shown.status}`}
        />
      )}

      <DetailList items={facts} />

      <div className="mt-6">
        <PanelHeading>Positionen</PanelHeading>
        <ul className="divide-y divide-line border-y border-line">
          {shown.items.map((item) => (
            <li key={item.position} className="flex items-baseline gap-3 py-2 text-body">
              <span className="flex-1 min-w-0 truncate">{item.description}</span>
              <span className="num text-ink-muted shrink-0">
                {`${formatQuantity(item.quantityMilli)} × ${formatCents(item.unitPrice, shown.currency)}`}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </DetailPanel>
  );
};
