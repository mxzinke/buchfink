import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Download, Plus } from 'lucide-react';
import type { JournalEntry, Receipt, ReceiptFileInput, ReceiptKind, ValidationFinding } from '../../types';
import { useNavigation, useRouteObject } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { useLiveReload } from '../../live';
import { Api } from '../../services/api';
import { usePostingLock } from '../../components/WriteLock';
import { ActorMark } from '../../components/ActorMark';
import { formatCents, formatDate, formatDateTime } from '../../utils/formatters';
import {
  Button,
  DetailList,
  DetailPanel,
  EmptyState,
  Notice,
  RowButton,
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
  toast,
  type Status,
} from '../../components/ui';
import { DocumentPreview, PanelHeading, errorText, grossOf } from './shared';
import { ReceiptBookingDialog } from './ReceiptBookingDialog';

const KIND_LABELS: Record<ReceiptKind, string> = {
  invoice: 'Rechnung',
  ticket: 'Fahrausweis',
  statement: 'Kontoauszug',
  self_issued: 'Eigenbeleg',
  letter: 'Brief',
  other: 'Sonstiger Beleg',
};

const ROLE_LABELS: Record<string, string> = {
  original: 'Original',
  structured: 'Rechnungsdatensatz',
  rendering: 'Darstellung',
  attachment: 'Anhang',
};

/** Ein abgelegter Beleg ist offen, ein verworfener zurückgenommen (§11.3). */
const STATUS: Record<Receipt['status'], Status> = {
  filed: 'offen',
  sealed: 'gebucht',
  assigned: 'zugeordnet',
  discarded: 'storniert',
};

type Filter = 'all' | 'open' | 'unclear';

const kindOf = (receipt: Receipt): ReceiptKind => receipt.kind ?? 'invoice';

/** Ob der Beleg noch gebucht werden muss, sagt das Backend. */
export const needsBooking = (receipt: Receipt) => receipt.needsBooking;

/**
 * Zu klären ist ein Beleg, dessen Rechnungsdatensatz gegen die Regeln verstößt
 * oder sich nicht lesen lässt. Das gilt auch nach dem Buchen: der
 * Vorsteuerabzug richtet sich nach der Rechnung.
 */
const needsClarification = (receipt: Receipt) =>
  receipt.status !== 'discarded' &&
  (receipt.validationErrors > 0 ||
    (receipt.files.some((f) => f.role === 'structured') && !receipt.detectedFormat));

function findingsOf(receipt: Receipt): ValidationFinding[] {
  try {
    const parsed = receipt.validationFindings ? JSON.parse(receipt.validationFindings) : [];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function matches(receipt: Receipt, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return [receipt.receiptNumber, receipt.issuerName, receipt.subject]
    .filter(Boolean)
    .some((text) => (text as string).toLowerCase().includes(q));
}

/**
 * Die Belege: was zu buchen und was zu klären ist, darunter alles andere. Der
 * Beleg selbst, seine Angaben und seine Buchung stehen in der Seitenleiste.
 * Gebucht wird in der App nur über eine Buchungsgruppe; für alles Weitere
 * (Bewirtung, Geschenke, Anzahlungen, Fremdwährung) ist Claude da.
 */
export const ReceiptsTab: React.FC = () => {
  const { year } = useWorkspace();
  const { id, open, close } = useRouteObject('books', 'receipts');
  const lock = usePostingLock();
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>('all');
  const [search, setSearch] = useState('');
  const [filing, setFiling] = useState(false);
  const [booking, setBooking] = useState<Receipt | null>(null);

  const load = useCallback(async () => {
    try {
      const [list, journal] = await Promise.all([Api.getReceipts(''), Api.getJournalEntries()]);
      // Was zu buchen ist, steht oben; sonst der neueste Beleg zuerst.
      setReceipts(
        [...(list ?? [])].sort(
          (a, b) =>
            Number(needsBooking(b)) - Number(needsBooking(a)) ||
            b.receiptNumber.localeCompare(a.receiptNumber),
        ),
      );
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

  useLiveReload(load, ['belege', 'buchungen', 'eingang']);

  /** Wer den Beleg gebucht hat, steht an seiner Buchung. */
  const entryOf = useMemo(() => {
    const map = new Map<number, JournalEntry>();
    for (const entry of entries) {
      if (entry.receiptId && entry.kind !== 'reversal' && !map.has(entry.receiptId)) {
        map.set(entry.receiptId, entry);
      }
    }
    return map;
  }, [entries]);

  const openCount = receipts.filter(needsBooking).length;
  const unclearCount = receipts.filter(needsClarification).length;
  const visible = useMemo(() => {
    const byFilter =
      filter === 'open'
        ? receipts.filter(needsBooking)
        : filter === 'unclear'
          ? receipts.filter(needsClarification)
          : receipts;
    return byFilter.filter((r) => matches(r, search.trim()));
  }, [receipts, filter, search]);

  const selected = id ? receipts.find((r) => String(r.id) === id) : undefined;

  async function fileReceipt() {
    setFiling(true);
    try {
      const paths = await Api.selectReceiptFiles();
      if (paths.length === 0) return;
      // Die erste Datei ist die empfangene Form, weitere sind Anhänge.
      const files: ReceiptFileInput[] = paths.map((path, index) => ({
        path,
        role: index === 0 ? 'original' : 'attachment',
      }));
      const receipt = await Api.fileIncomingReceipt(files);
      toast.success(`Beleg ${receipt.receiptNumber} abgelegt.`);
      await load();
      open(String(receipt.id));
    } catch (e) {
      toast.error(errorText(e));
    } finally {
      setFiling(false);
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Beleg, Aussteller oder Betreff"
          aria-label="Belege durchsuchen"
          className="max-w-sm"
        />
        <Select<Filter>
          className="w-52"
          aria-label="Belege filtern"
          items={[
            { value: 'all', label: `Alle Belege (${receipts.length})` },
            { value: 'open', label: `Zu buchen (${openCount})` },
            { value: 'unclear', label: `Zu klären (${unclearCount})` },
          ]}
          value={filter}
          onValueChange={setFilter}
        />
        <div className="ml-auto">
          <Button
            variant="secondary"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            loading={filing}
            disabled={lock.locked}
            title={lock.hint ?? 'Eine Belegdatei von diesem Rechner ablegen'}
            onClick={() => void fileReceipt()}
          >
            Beleg ablegen
          </Button>
        </div>
      </div>

      <div className="mt-4">
        {error ? (
          <Notice tone="negative" text={error} />
        ) : loading ? (
          <SkeletonRows rows={8} />
        ) : receipts.length === 0 ? (
          <EmptyState
            title="Noch keine Belege"
            description="Claude legt Belege für Sie ab. Einzelne Dateien legen Sie hier ab."
          />
        ) : visible.length === 0 ? (
          <EmptyState
            variant="gefiltert"
            title="Kein Beleg passt zum Filter"
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
                <Th className="w-32">Beleg</Th>
                <Th optional className="w-28">
                  Belegdatum
                </Th>
                <Th>Aussteller und Betreff</Th>
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
              {visible.map((receipt) => {
                const entry = receipt.status === 'sealed' ? entryOf.get(receipt.id) : undefined;
                const unclear = needsClarification(receipt);
                return (
                  <Tr
                    key={receipt.id}
                    variant={
                      receipt.status === 'discarded'
                        ? 'storno'
                        : String(receipt.id) === id
                          ? 'selected'
                          : 'default'
                    }
                    className="cursor-pointer"
                    onClick={() => open(String(receipt.id))}
                  >
                    <Td code>
                      <RowButton onOpen={() => open(String(receipt.id))}>{receipt.receiptNumber}</RowButton>
                    </Td>
                    <Td optional className="text-ink-subtle num">
                      {receipt.documentDate ? formatDate(receipt.documentDate) : '—'}
                    </Td>
                    <Td fill>
                      {receipt.issuerName || KIND_LABELS[kindOf(receipt)]}
                      {receipt.subject && <span className="text-ink-subtle"> · {receipt.subject}</span>}
                    </Td>
                    <Td numeric>
                      {receipt.grossAmount !== undefined
                        ? formatCents(receipt.grossAmount, receipt.currency || 'EUR')
                        : '—'}
                    </Td>
                    <Td className="px-2">
                      {entry && <ActorMark actor={entry.actor} channel={entry.channel} />}
                    </Td>
                    <Td>
                      {unclear && receipt.status === 'filed' ? (
                        <StatusBadge status="fehlerhaft" />
                      ) : receipt.status === 'filed' && !receipt.needsBooking ? (
                        <StatusBadge status="abgelegt" />
                      ) : (
                        <StatusBadge status={STATUS[receipt.status]} />
                      )}
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </div>

      <ReceiptPanel receipt={selected} onClose={close} onBook={(receipt) => setBooking(receipt)} />

      {booking && (
        <ReceiptBookingDialog
          receipt={booking}
          onClose={() => setBooking(null)}
          onBooked={async (entry) => {
            setBooking(null);
            toast.success(`Beleg gebucht als ${entry.entryNumber}.`);
            await load();
          }}
        />
      )}
    </div>
  );
};

/** Die Seitenleiste eines Belegs: das Dokument, seine Angaben, seine Buchung. */
const ReceiptPanel: React.FC<{
  receipt: Receipt | undefined;
  onClose: () => void;
  onBook: (receipt: Receipt) => void;
}> = ({ receipt, onClose, onBook }) => {
  const lock = usePostingLock();
  const [shown, setShown] = useState<Receipt | undefined>(receipt);
  useEffect(() => {
    if (receipt) setShown(receipt);
  }, [receipt]);
  if (!shown) return null;

  const kind = kindOf(shown);
  const bookable = needsBooking(shown);
  const findings = findingsOf(shown).filter((f) => f.severity === 'fatal');

  const facts = [
    { label: 'Belegart', value: KIND_LABELS[kind] },
    {
      label: 'Belegdatum',
      value: shown.documentDate ? formatDate(shown.documentDate) : 'fehlt',
      numeric: true,
    },
    { label: 'Aussteller', value: shown.issuerName || 'fehlt' },
    ...(shown.grossAmount !== undefined
      ? [
          {
            label: 'Betrag',
            value: `${formatCents(shown.grossAmount, shown.currency || 'EUR')}${
              shown.taxAmount
                ? ` · darin Steuer ${formatCents(shown.taxAmount, shown.currency || 'EUR')}`
                : ''
            }`,
            numeric: true,
          },
        ]
      : []),
    {
      label: 'Eingang',
      value: `${shown.receivedAt ? formatDate(shown.receivedAt) : formatDateTime(shown.createdAt)}${
        shown.receivedVia ? ` · ${shown.receivedVia}` : ''
      }`,
    },
    ...(shown.detectedFormat
      ? [
          {
            label: 'E-Rechnung',
            value: [shown.detectedFormat, shown.detectedProfile].filter(Boolean).join(' · '),
          },
        ]
      : []),
    ...(shown.retentionUntil
      ? [{ label: 'Aufbewahren bis', value: formatDate(shown.retentionUntil), numeric: true }]
      : []),
    ...(shown.discardReason ? [{ label: 'Verworfen', value: shown.discardReason }] : []),
    ...(shown.assignedKind === 'shareholder_loan'
      ? [{ label: 'Zugeordnet', value: `Vertrag zum Gesellschafterdarlehen Nr. ${shown.assignedId}` }]
      : []),
  ];

  return (
    <DetailPanel
      open={Boolean(receipt)}
      onOpenChange={(next) => !next && onClose()}
      title={shown.receiptNumber}
      context={[shown.issuerName, shown.subject].filter(Boolean).join(' · ') || KIND_LABELS[kind]}
      meta={<StatusBadge status={STATUS[shown.status]} />}
      width="lg"
      footer={
        bookable ? (
          <Button variant="primary" disabled={lock.locked} title={lock.hint} onClick={() => onBook(shown)}>
            Beleg buchen
          </Button>
        ) : undefined
      }
    >
      <DocumentPreview
        load={() => Api.getReceiptPreview(shown.id)}
        loadKey={`${shown.id}-${shown.receiptHash}`}
        fallback={
          kind === 'statement'
            ? 'Dieser Kontoauszug ist eine XML-Datei. Seine Umsätze stehen unter Bank.'
            : undefined
        }
      />

      {findings.length > 0 && (
        <Notice
          className="mt-4"
          tone="negative"
          text={`Der Rechnungsdatensatz verstößt gegen ${findings.length} ${
            findings.length === 1 ? 'Regel' : 'Regeln'
          }: ${findings.map((f) => f.message).join(' ')}`}
        />
      )}

      <DetailList className="mt-6" items={facts} />

      <ReceiptEntries receiptId={shown.id} status={shown.status} />

      <ReceiptFiles receipt={shown} />
    </DetailPanel>
  );
};

/** Die Buchungen zum Beleg; jede führt ins Journal. */
const ReceiptEntries: React.FC<{ receiptId: number; status: Receipt['status'] }> = ({
  receiptId,
  status,
}) => {
  const { navigate } = useNavigation();
  const [entries, setEntries] = useState<JournalEntry[] | null>(null);

  useEffect(() => {
    if (status === 'filed') {
      setEntries([]);
      return;
    }
    let cancelled = false;
    Api.getEntriesForReceipt(receiptId)
      .then((list) => !cancelled && setEntries(list))
      .catch(() => !cancelled && setEntries([]));
    return () => {
      cancelled = true;
    };
  }, [receiptId, status]);

  if (!entries || entries.length === 0) return null;
  return (
    <div className="mt-6">
      <PanelHeading>Buchung</PanelHeading>
      <ul className="divide-y divide-line border-y border-line">
        {entries.map((entry) => (
          <li key={entry.id} className="flex items-center gap-3 py-2 text-body">
            <button
              type="button"
              className="code-num text-caption text-accent-text hover:underline w-28 text-left shrink-0"
              onClick={() => navigate({ area: 'books', tab: 'journal', id: entry.entryNumber })}
            >
              {entry.entryNumber}
            </button>
            <span className="num text-ink-subtle w-24 shrink-0">{formatDate(entry.bookingDate)}</span>
            <span className="flex-1 truncate">{entry.description}</span>
            <span className="num">{formatCents(grossOf(entry), entry.currency)}</span>
            <ActorMark actor={entry.actor} channel={entry.channel} />
          </li>
        ))}
      </ul>
    </div>
  );
};

/** Die Dateien des Belegs. Speichern gibt eine Kopie heraus, der Beleg bleibt im Archiv. */
const ReceiptFiles: React.FC<{ receipt: Receipt }> = ({ receipt }) => {
  const [busy, setBusy] = useState<number | null>(null);

  async function save(fileId: number) {
    setBusy(fileId);
    try {
      const path = await Api.saveReceiptFileAs(receipt.id, fileId);
      if (path) toast.success(`Gespeichert: ${path}`);
    } catch (e) {
      toast.error(errorText(e));
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="mt-6">
      <PanelHeading>Dateien</PanelHeading>
      <ul className="divide-y divide-line border-y border-line">
        {receipt.files.map((file) => (
          <li key={file.id} className="flex items-center gap-3 py-1.5 text-body">
            <span className="flex-1 truncate">{file.fileName}</span>
            <span className="text-caption text-ink-subtle">
              {ROLE_LABELS[file.role] ?? file.role}
              {file.derived ? ' · erzeugt' : ''}
            </span>
            <Button
              variant="quiet"
              size="sm"
              iconOnly
              loading={busy === file.id}
              title="Kopie speichern"
              aria-label={`${file.fileName} speichern`}
              onClick={() => void save(file.id)}
            >
              <Download className="w-4 h-4" strokeWidth={1.5} />
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
};
