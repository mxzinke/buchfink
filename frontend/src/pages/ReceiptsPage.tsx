import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  FileCode,
  FileText,
  Paperclip,
  Plus,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import type {
  Account,
  Contact,
  EInvoiceProposal,
  EntertainmentDetail,
  PostingGroup,
  PostingPreview,
  PostingWarning,
  Receipt,
  ReceiptFileInput,
  ReceiptFileRole,
  ReceiptRequest,
  Settlement,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
  ValidationFinding,
} from '../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, parseCents } from '../utils/formatters';
import {
  Button,
  Combobox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Tr,
  cn,
  toast,
  type Status,
} from '../components/ui';

/**
 * Belege.
 *
 * Ablegen und Buchen sind zwei Schritte. Eine XRechnung muss sofort ablegbar
 * sein — die GoBD verlangen die Aufbewahrung in der empfangenen Form —, aber sie
 * darf nicht buchbar sein, bevor eine Darstellung existiert: niemand soll eine
 * Buchung zu einem Beleg freigeben, den er nicht ansehen kann.
 *
 * Diese Seite rechnet nichts. Der Buchungssatz samt Netto, Steuer und Brutto
 * kommt aus der Vorschau des Backends.
 */

const ROLE_LABELS: Record<ReceiptFileRole, string> = {
  original: 'Original',
  structured: 'Strukturierter Teil',
  rendering: 'Darstellung',
  attachment: 'Anhang',
};

/**
 * Das Status-Vokabular ist abgeschlossen (§11.3). Ein abgelegter Beleg ist ein
 * offener Vorgang, ein verworfener ist zurückgenommen — dafür gibt es keine
 * eigenen Wörter, und es sollen auch keine entstehen.
 */
const STATUS: Record<Receipt['status'], Status> = {
  filed: 'offen',
  sealed: 'gebucht',
  discarded: 'storniert',
};

/** Hinweisfläche nach §6.2, Fall 4. Trägt Rand und Fläche immer zusammen. */
const NOTE = 'rounded-control border px-4 py-3';
const NOTE_TONE = {
  neutral: 'border-line-strong bg-sunken',
  attention: 'border-attention-line bg-attention-soft',
  positive: 'border-positive-line bg-positive-soft',
  negative: 'border-negative-line bg-negative-soft',
};

interface DraftPosition {
  postingGroup: string;
  net: string;
  taxRate: TaxRate;
  text: string;
}

const emptyPosition = (group?: PostingGroup): DraftPosition => ({
  postingGroup: group?.key ?? '',
  net: '',
  taxRate: group?.defaultRate ?? TAX_RATE_STANDARD,
  text: '',
});

export const ReceiptsPage: React.FC = () => {
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [groups, setGroups] = useState<PostingGroup[]>([]);
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [selected, setSelected] = useState<Receipt | null>(null);
  const [proposal, setProposal] = useState<EInvoiceProposal | null>(null);
  const [proposalError, setProposalError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [filing, setFiling] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [list, contactList, groupList, treatmentList, accounts] = await Promise.all([
        Api.getReceipts(''),
        Api.getContacts(),
        Api.getPostingGroups('incoming'),
        Api.getTaxTreatments('incoming'),
        Api.getPaymentAccounts(),
      ]);
      setReceipts(list ?? []);
      setContacts(contactList ?? []);
      setGroups(groupList ?? []);
      setTreatments(treatmentList ?? []);
      setPaymentAccounts(accounts ?? []);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // Liegt ein strukturierter Teil vor, ist er die Buchungsquelle — der
  // Vorsteuerabzug ist nur aus ihm möglich (UStAE 14c.1 Abs. 4a Satz 4).
  useEffect(() => {
    setProposalError(null);
    if (!selected || selected.status !== 'filed') {
      setProposal(null);
      return;
    }
    if (!selected.files.some((f) => f.role === 'structured')) {
      setProposal(null);
      return;
    }
    let cancelled = false;
    Api.proposeFromEInvoice(selected.id)
      .then((p) => {
        if (!cancelled) setProposal(p);
      })
      .catch((e) => {
        // Kein Toast: warum aus diesem Datensatz kein Vorschlag wird — eine
        // Gutschrift, ein unbekannter Steuerkategoriecode —, muss stehen
        // bleiben, solange der Beleg offen ist. Eine Meldung, die nach fünf
        // Sekunden verschwindet, beantwortet die Frage genau einmal.
        if (!cancelled) {
          setProposal(null);
          setProposalError(e instanceof Error ? e.message : String(e));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [selected]);

  // TODO: Drag & Drop über den Wails-Drop-Handler. Er liefert wie der Dialog
  // Pfade, sodass mehrere Megabyte große Scans nicht über die IPC-Grenze
  // müssen. Heute führt der einzige Weg über den Knopf.
  async function fileReceipt() {
    setFiling(true);
    try {
      const paths = await Api.selectReceiptFiles();
      if (!paths || paths.length === 0) return;

      // Die erste Datei ist die empfangene Form, weitere sind Anhänge. Rollen
      // lassen sich am abgelegten Beleg noch ändern.
      const files: ReceiptFileInput[] = paths.map((path, index) => ({
        path,
        role: index === 0 ? 'original' : 'attachment',
      }));
      const receipt = await Api.fileIncomingReceipt(files);
      toast.success(`Beleg ${receipt.receiptNumber} abgelegt.`);
      await load();
      setSelected(receipt);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setFiling(false);
    }
  }

  const vendors = useMemo(() => contacts.filter((c) => c.type === 'vendor'), [contacts]);
  const openCount = receipts.filter((r) => r.status === 'filed').length;

  return (
    <div className="max-w-[1440px] mx-auto px-8 py-8">
      <PageHeader
        title="Belege"
        context={loading ? undefined : `${receipts.length} abgelegt · ${openCount} noch zu buchen`}
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zum Ablegen und Buchen">
              Belege werden zuerst abgelegt und dann gebucht. Ein Beleg kann aus mehreren Dateien
              bestehen: Eine ZUGFeRD-Rechnung ist ein PDF mit eingebettetem XML, eine XRechnung ist
              reines XML und braucht vor dem Buchen eine erzeugte Darstellung.
            </HelpPopover>
            <Button
              variant="secondary"
              disabled={loading}
              onClick={() => void load()}
              icon={<RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} strokeWidth={1.5} />}
            >
              Aktualisieren
            </Button>
            <Button
              variant="primary"
              loading={filing}
              onClick={() => void fileReceipt()}
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            >
              Beleg ablegen
            </Button>
          </div>
        }
      />

      <div className="mt-6 grid grid-cols-1 xl:grid-cols-[19rem_minmax(0,1fr)] gap-8">
        <aside className="xl:border-r xl:border-line xl:pr-6">
          <ReceiptList
            receipts={receipts}
            loading={loading}
            selectedId={selected?.id}
            onSelect={setSelected}
          />
        </aside>

        {selected ? (
          <ReceiptDetail
            key={selected.id}
            receipt={selected}
            vendors={vendors}
            groups={groups}
            treatments={treatments}
            paymentAccounts={paymentAccounts}
            proposal={proposal}
            proposalError={proposalError}
            onChanged={async (updated) => {
              setSelected(updated);
              await load();
            }}
            onBooked={async (entryNumber) => {
              toast.success(`Beleg gebucht als ${entryNumber}.`);
              setSelected(null);
              await load();
            }}
          />
        ) : (
          <EmptyState
            icon={<FileText className="w-6 h-6" strokeWidth={1.5} />}
            title="Kein Beleg ausgewählt"
            description="Links einen Beleg wählen, um ihn anzusehen und zu buchen."
          />
        )}
      </div>
    </div>
  );
};

// -------------------------------------------------------------------------

const ReceiptList: React.FC<{
  receipts: Receipt[];
  loading: boolean;
  selectedId?: number;
  onSelect: (r: Receipt) => void;
}> = ({ receipts, loading, selectedId, onSelect }) => {
  if (loading) return <SkeletonRows rows={6} />;
  if (receipts.length === 0) return <EmptyState title="Noch keine Belege abgelegt" />;

  return (
    <nav className="flex flex-col gap-0.5" aria-label="Belege">
      {receipts.map((receipt) => {
        const active = selectedId === receipt.id;
        const original = receipt.files.find((f) => f.role === 'original');
        return (
          <button
            key={receipt.id}
            type="button"
            aria-current={active || undefined}
            onClick={() => onSelect(receipt)}
            className={cn(
              'flex items-start gap-2.5 rounded-card px-3 py-2.5 text-left',
              'transition-colors duration-120 ease-quiet',
              active ? 'bg-accent-soft' : 'hover:bg-sunken',
            )}
          >
            {/* Einseitige Markierung als Pille, nicht als gekrümmte Border (§12). */}
            <span
              className={cn(
                'mt-0.5 h-8 w-0.5 shrink-0 rounded-full',
                active ? 'bg-accent' : 'bg-transparent',
              )}
            />
            <span className="min-w-0 flex-1">
              <span className="flex items-center justify-between gap-2">
                <span className="code-num text-caption text-ink">{receipt.receiptNumber}</span>
                <StatusBadge status={STATUS[receipt.status]} />
              </span>
              <span className="block text-caption text-ink-muted truncate mt-1">
                {original?.fileName ?? '—'}
              </span>
              <span className="flex items-center gap-2 text-caption text-ink-subtle mt-0.5">
                <span className="num">
                  {receipt.receivedAt ? formatDate(receipt.receivedAt) : '—'}
                </span>
                {receipt.files.length > 1 && (
                  <span className="inline-flex items-center gap-1">
                    <Paperclip className="w-3 h-3" strokeWidth={1.5} />
                    {receipt.files.length}
                  </span>
                )}
              </span>
            </span>
          </button>
        );
      })}
    </nav>
  );
};

// -------------------------------------------------------------------------

const ReceiptDetail: React.FC<{
  receipt: Receipt;
  vendors: Contact[];
  groups: PostingGroup[];
  treatments: TaxTreatmentInfo[];
  paymentAccounts: Account[];
  onChanged: (updated: Receipt) => Promise<void>;
  onBooked: (entryNumber: string) => Promise<void>;
  proposal: EInvoiceProposal | null;
  proposalError: string | null;
}> = ({
  receipt,
  vendors,
  groups,
  treatments,
  paymentAccounts,
  proposal,
  proposalError,
  onChanged,
  onBooked,
}) => (
  <div className="grid grid-cols-1 2xl:grid-cols-2 gap-8 items-start">
    <ReceiptViewer receipt={receipt} onChanged={onChanged} />

    {receipt.status === 'filed' ? (
      <div>
        <ProposalRefusal message={proposalError} />
        <BookingForm
          key={proposal ? `proposal-${receipt.id}` : `blank-${receipt.id}`}
          receipt={receipt}
          vendors={vendors}
          groups={groups}
          treatments={treatments}
          paymentAccounts={paymentAccounts}
          proposal={proposal}
          onBooked={onBooked}
        />
      </div>
    ) : (
      <div>
        <h2 className="flex items-center gap-2 text-heading text-ink">
          <ShieldCheck className="w-4 h-4 shrink-0 text-positive" strokeWidth={1.5} />
          {receipt.status === 'sealed' ? 'Gebucht und versiegelt' : 'Verworfen'}
        </h2>
        {receipt.status === 'sealed' && (
          <p className="text-body text-ink-muted mt-2">
            Was später dazukommt — eine Mahnung, ein Zahlungsnachweis, eine korrigierte Rechnung —
            ist ein eigener Beleg auf denselben Geschäftsvorfall. Eine inhaltliche Korrektur läuft
            über den Storno der Buchung.
          </p>
        )}
        {receipt.discardReason && (
          <p className="text-body text-ink-muted mt-2">Grund: {receipt.discardReason}</p>
        )}
      </div>
    )}
  </div>
);

// -------------------------------------------------------------------------

const ReceiptViewer: React.FC<{
  receipt: Receipt;
  onChanged: (updated: Receipt) => Promise<void>;
}> = ({ receipt, onChanged }) => {
  const [preview, setPreview] = useState<{ dataUrl: string; mimeType: string; intact: boolean } | null>(
    null,
  );
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [discarding, setDiscarding] = useState(false);
  const [discardReason, setDiscardReason] = useState('');

  useEffect(() => {
    let cancelled = false;
    setPreview(null);
    setPreviewError(null);
    Api.getReceiptPreview(receipt.id)
      .then((p) => {
        if (!cancelled) setPreview(p);
      })
      .catch((e) => {
        if (!cancelled) setPreviewError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [receipt.id, receipt.receiptHash]);

  async function addFile(role: ReceiptFileRole) {
    setBusy(true);
    try {
      const paths = await Api.selectReceiptFiles(
        role === 'rendering' ? 'Darstellung auswählen' : 'Anhang auswählen',
      );
      if (!paths || paths.length === 0) return;
      let updated = receipt;
      for (const path of paths) {
        updated = await Api.addReceiptFile(receipt.id, { path, role });
      }
      await onChanged(updated);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function removeFile(fileId: number) {
    setBusy(true);
    try {
      await onChanged(await Api.removeReceiptFile(receipt.id, fileId));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  /**
   * Zieht den strukturierten Rechnungsdatensatz aus dem Beleg.
   *
   * Ein eigener Schritt: der Beleg wird in der empfangenen Form abgelegt und
   * erst danach untersucht. Findet sich nichts, ist es eine sonstige Rechnung —
   * die Meldung sagt das.
   */
  async function extractStructured() {
    setBusy(true);
    try {
      await onChanged(await Api.extractStructuredPart(receipt.id));
      toast.success('Strukturierter Rechnungsdatensatz übernommen.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function discard() {
    const reason = discardReason.trim();
    if (!reason) return;
    setDiscarding(false);
    setBusy(true);
    try {
      await Api.discardReceipt(receipt.id, reason);
      await onChanged({ ...receipt, status: 'discarded', discardReason: reason });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const open = receipt.status === 'filed';

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <h2 className="code-num text-heading text-ink">{receipt.receiptNumber}</h2>
          <p className="code-num text-caption text-ink-subtle mt-0.5">
            Beleg-Hash {receipt.receiptHash.slice(0, 12)}…
          </p>
        </div>
        {open && (
          <Button variant="quiet" size="sm" disabled={busy} onClick={() => setDiscarding(true)}>
            Verwerfen
          </Button>
        )}
      </div>

      {/* Der Beleg ist ein Fremdkörper in der Oberfläche und bekommt deshalb
          eine eigene Fläche (§6.2, Fall 3). */}
      <div className="mt-4 rounded-card border border-line bg-sunken min-h-[20rem] flex items-center justify-center p-4">
        {previewError ? (
          <div className="text-center max-w-sm">
            <AlertTriangle className="w-6 h-6 mx-auto text-attention" strokeWidth={1.5} />
            <p className="text-body text-ink-muted mt-3">{previewError}</p>
            {open && (
              <Button
                variant="secondary"
                className="mt-4"
                disabled={busy}
                onClick={() => void addFile('rendering')}
              >
                Darstellung hinzufügen
              </Button>
            )}
          </div>
        ) : !preview ? (
          <span className="text-body text-ink-subtle">Vorschau wird geladen …</span>
        ) : preview.mimeType === 'application/pdf' ? (
          <iframe
            title="Belegvorschau"
            src={preview.dataUrl}
            className="w-full h-[30rem] rounded-control bg-surface"
          />
        ) : (
          <img src={preview.dataUrl} alt="Beleg" className="max-h-[30rem] rounded-control" />
        )}
      </div>

      {preview && !preview.intact && (
        <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text mt-3')}>
          Die Datei auf der Platte passt nicht mehr zu ihrer Prüfsumme. Sie wurde nach dem Ablegen
          verändert.
        </p>
      )}

      <ValidationPanel receipt={receipt} />

      <ul className="mt-4 divide-y divide-line border-t border-line">
        {receipt.files.map((file) => (
          <li key={file.id} className="flex items-center gap-2 py-2 text-body">
            <FileText className="w-4 h-4 shrink-0 text-ink-faint" strokeWidth={1.5} />
            <span className="truncate flex-1 text-ink">{file.fileName}</span>
            <span className="shrink-0 text-caption text-ink-subtle">
              {ROLE_LABELS[file.role]}
              {file.derived && ' · erzeugt'}
            </span>
            {open && receipt.files.length > 1 && file.role !== 'original' && (
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                disabled={busy}
                title="Datei entfernen"
                aria-label={`${file.fileName} entfernen`}
                onClick={() => void removeFile(file.id)}
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </Button>
            )}
          </li>
        ))}
      </ul>

      {open && (
        <div className="mt-4 flex flex-wrap gap-2">
          {!receipt.files.some((f) => f.role === 'structured') && (
            <Button
              variant="secondary"
              size="sm"
              disabled={busy}
              onClick={() => void extractStructured()}
              icon={<FileCode className="w-3.5 h-3.5" strokeWidth={1.5} />}
            >
              E-Rechnung auslesen
            </Button>
          )}
          <Button variant="secondary" size="sm" disabled={busy} onClick={() => void addFile('attachment')}>
            Anhang hinzufügen
          </Button>
          {!receipt.files.some((f) => f.role === 'rendering') && (
            <Button
              variant="secondary"
              size="sm"
              disabled={busy}
              onClick={() => void addFile('rendering')}
            >
              Darstellung hinzufügen
            </Button>
          )}
        </div>
      )}

      <Dialog
        open={discarding}
        onOpenChange={(next) => {
          setDiscarding(next);
          if (!next) setDiscardReason('');
        }}
        title="Beleg verwerfen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDiscarding(false)}>
              Abbrechen
            </Button>
            <Button variant="danger" disabled={!discardReason.trim()} onClick={() => void discard()}>
              Verwerfen
            </Button>
          </>
        }
      >
        <p className="text-body text-ink-muted">
          <span className="code-num text-ink">{receipt.receiptNumber}</span> bleibt aufbewahrt und
          nachvollziehbar, wird aber nicht gebucht.
        </p>
        <Field label="Grund" className="mt-4">
          <Input
            value={discardReason}
            onChange={(e) => setDiscardReason(e.target.value)}
            placeholder="Doppelt erhalten"
          />
        </Field>
      </Dialog>
    </div>
  );
};

// -------------------------------------------------------------------------

const POSITION_GRID = 'grid grid-cols-[minmax(0,1fr)_7rem_6rem_2rem] gap-2 items-start';

const BookingForm: React.FC<{
  receipt: Receipt;
  vendors: Contact[];
  groups: PostingGroup[];
  treatments: TaxTreatmentInfo[];
  paymentAccounts: Account[];
  proposal: EInvoiceProposal | null;
  onBooked: (entryNumber: string) => Promise<void>;
}> = ({ receipt, vendors, groups, treatments, paymentAccounts, proposal, onBooked }) => {
  const today = receipt.receivedAt || new Date().toISOString().split('T')[0];
  const p = proposal?.request;
  const [contactId, setContactId] = useState(p?.contactId || vendors[0]?.id || 0);
  const [documentDate, setDocumentDate] = useState(p?.documentDate || today);
  const [bookingDate, setBookingDate] = useState(p?.bookingDate || today);
  const [serviceFrom, setServiceFrom] = useState(p?.serviceDateFrom || today);
  const [serviceTo, setServiceTo] = useState(p?.serviceDateTo || today);
  // Ohne Vorschlag steht „Inland" — die gewöhnliche Eingangsrechnung. Kam ein
  // Vorschlag, ließ sich der Steuerfall daraus aber nicht ableiten, bleibt das
  // Feld leer: das Backend hat sich bewusst enthalten (gemischte Kategorien, ein
  // Code ohne deutsche Entsprechung), und „Inland" hier einzusetzen überschriebe
  // genau diese Enthaltung. Das Backend nimmt eine Buchung ohne Steuerfall nicht an.
  const [treatment, setTreatment] = useState<TaxTreatment | ''>(
    proposal ? p?.taxTreatment || '' : 'domestic',
  );
  const [positions, setPositions] = useState<DraftPosition[]>(
    // Beträge und Steuersätze kommen aus dem strukturierten Teil, die
    // Buchungsgruppe bleibt offen — sie steht in keiner Rechnung.
    p && p.positions.length > 0
      ? p.positions.map((pos) => ({
          postingGroup: '',
          net: (pos.net / 100).toFixed(2).replace('.', ','),
          taxRate: pos.taxRate,
          text: pos.text ?? '',
        }))
      : [emptyPosition(groups[0])],
  );
  const [settlement, setSettlement] = useState<Settlement>('open');
  const [paymentAccount, setPaymentAccount] = useState(paymentAccounts[0]?.number ?? '');
  const [description, setDescription] = useState(p?.description ?? '');
  const [entertainment, setEntertainment] = useState<EntertainmentDetail>({
    place: '',
    day: today,
    participants: '',
    occasion: '',
  });

  // Ein Betrag, den parseCents nicht lesen kann ("1.2.3", "250,--", auch das
  // deutsche "1.234"), fällt beim Aufbau der Anfrage unten aus den Positionen
  // heraus. Ohne diese Prüfung würde der Beleg ohne diese Position gebucht, und
  // die Vorschau sähe dabei sauber aus. Das ist keine Fachlogik: gerechnet wird
  // nichts, es wird nur gesagt, dass hier etwas nicht lesbar ist.
  const unreadableAmount = (value: string) => value.trim() !== '' && parseCents(value) === null;
  const hasUnreadableAmount = positions.some((pos) => unreadableAmount(pos.net));

  // Ob die Aufzeichnung nötig ist, sagt der Katalog: die Gruppe trägt das Konto
  // für den nicht abzugsfähigen Anteil. Das Backend besteht darauf.
  const needsEntertainment = positions.some(
    (pos) => groups.find((g) => g.key === pos.postingGroup)?.deductibleQuota === 'entertainment',
  );

  const [preview, setPreview] = useState<PostingPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const request: ReceiptRequest = useMemo(
    () => ({
      contactId,
      receiptId: receipt.id,
      bookingDate,
      documentDate,
      serviceDateFrom: serviceFrom,
      serviceDateTo: serviceTo,
      description,
      taxTreatment: treatment as TaxTreatment,
      positions: positions
        .filter((pos) => pos.postingGroup && parseCents(pos.net) !== null)
        .map((pos) => ({
          postingGroup: pos.postingGroup,
          net: parseCents(pos.net) ?? 0,
          taxRate: pos.taxRate,
          text: pos.text || undefined,
        })),
      settlement,
      paymentAccount: settlement === 'paid' ? paymentAccount : undefined,
      currency: 'EUR',
      entertainment: needsEntertainment ? entertainment : undefined,
    }),
    [
      contactId,
      receipt.id,
      bookingDate,
      documentDate,
      serviceFrom,
      serviceTo,
      description,
      treatment,
      positions,
      settlement,
      paymentAccount,
      needsEntertainment,
      entertainment,
    ],
  );

  // Der Buchungssatz kommt aus dem Backend. Das Frontend rechnet ihn nicht nach:
  // eine zweite Steuerrechnung hier wäre eine zweite Wahrheit.
  useEffect(() => {
    if (!contactId || !treatment || request.positions.length === 0) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewIncomingReceipt(request)
        .then((result) => {
          if (cancelled) return;
          setPreview(result);
          setPreviewError(null);
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setPreviewError(e instanceof Error ? e.message : String(e));
        });
    }, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [request, contactId, treatment]);

  function updatePosition(index: number, patch: Partial<DraftPosition>) {
    setPositions((prev) => prev.map((pos, i) => (i === index ? { ...pos, ...patch } : pos)));
  }

  function selectGroup(index: number, key: string) {
    const group = groups.find((g) => g.key === key);
    updatePosition(index, { postingGroup: key, taxRate: group?.defaultRate ?? TAX_RATE_STANDARD });
    // Die Gruppe schlägt den Steuerfall vor — Miete ist steuerfrei, Gehälter sind
    // nicht steuerbar. Ein Satz von null allein sagt nicht, warum keine Steuer
    // anfällt, und das Backend besteht auf der Begründung.
    if (group?.defaultTreatment) setTreatment(group.defaultTreatment);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const entry = await Api.postIncomingReceipt(request);
      await onBooked(entry.entryNumber);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const groupItems = useMemo(
    () => groups.map((g) => ({ value: g.key, label: g.label, meta: g.category })),
    [groups],
  );

  if (vendors.length === 0) {
    return (
      <EmptyState
        title="Noch kein Lieferant angelegt"
        description="Für einen Eingangsbeleg braucht es einen Lieferanten. Unter Kunden & Lieferanten bekommt er sein Personenkonto aus dem Kreditoren-Nummernkreis."
      />
    );
  }

  return (
    <form onSubmit={submit}>
      <h2 className="text-heading text-ink">Beleg buchen</h2>

      <div className="mt-5 flex flex-col gap-4">
        <Field label="Lieferant">
          <Select
            items={vendors.map((v) => ({ value: v.id, label: `${v.name} · ${v.ledgerAccount}` }))}
            value={contactId}
            onValueChange={setContactId}
          />
        </Field>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Belegdatum" hint="Rechnungsdatum">
            <Input type="date" value={documentDate} onChange={(e) => setDocumentDate(e.target.value)} />
          </Field>
          <Field label="Buchungsdatum" hint="bestimmt die Periode">
            <Input type="date" value={bookingDate} onChange={(e) => setBookingDate(e.target.value)} />
          </Field>
          <Field label="Leistung von" hint="§ 14 Abs. 4 Nr. 6 UStG">
            <Input type="date" value={serviceFrom} onChange={(e) => setServiceFrom(e.target.value)} />
          </Field>
          <Field label="Leistung bis">
            <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
          </Field>
        </div>

        <Field
          label="Steuerfall"
          hint={treatments.find((t) => t.treatment === treatment)?.hint}
          help="Der Steuerfall entscheidet über Aufwandskonto und Steuerzeile. Ohne ihn nimmt Buchfink die Buchung nicht an."
        >
          <Select
            items={treatments.map((t) => ({ value: t.treatment, label: t.label }))}
            value={treatment || undefined}
            onValueChange={(next) => setTreatment(next as TaxTreatment)}
            placeholder="Steuerfall wählen"
          />
        </Field>
      </div>

      <div className="mt-6 pt-6 border-t border-line">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-label text-ink-muted">Positionen</h3>
          <Button
            variant="quiet"
            size="sm"
            icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
            onClick={() => setPositions((prev) => [...prev, emptyPosition(groups[0])])}
          >
            Position hinzufügen
          </Button>
        </div>

        <div className={cn(POSITION_GRID, 'text-caption text-ink-subtle mb-1')}>
          <span>Buchungsgruppe</span>
          <span className="text-right">Netto</span>
          <span>USt</span>
          <span />
        </div>

        <div className="flex flex-col gap-2">
          {positions.map((position, index) => (
            <div key={index} className={POSITION_GRID}>
              <Combobox
                items={groupItems}
                value={position.postingGroup || null}
                onValueChange={(key) => selectGroup(index, key ?? '')}
                placeholder="Gruppe suchen"
                emptyText="Keine passende Buchungsgruppe."
              />
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={position.net}
                onChange={(e) => updatePosition(index, { net: e.target.value })}
                aria-label={`Nettobetrag der Position ${index + 1}`}
                title={
                  unreadableAmount(position.net)
                    ? 'Der Betrag ist nicht lesbar. Erwartet wird etwa 1234,56 — ohne Tausenderpunkt.'
                    : undefined
                }
                className={cn(
                  unreadableAmount(position.net) && 'border-negative ring-2 ring-negative/20',
                )}
              />
              <Select
                items={[
                  { value: TAX_RATE_STANDARD, label: '19 %' },
                  { value: TAX_RATE_REDUCED, label: '7 %' },
                  { value: TAX_RATE_NONE, label: 'ohne' },
                ]}
                value={position.taxRate}
                onValueChange={(taxRate) => updatePosition(index, { taxRate })}
              />
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                disabled={positions.length === 1}
                title="Position entfernen"
                aria-label={`Position ${index + 1} entfernen`}
                onClick={() => setPositions((prev) => prev.filter((_, i) => i !== index))}
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </Button>
            </div>
          ))}
        </div>
      </div>

      <div className="mt-6 pt-6 border-t border-line flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Zahlung">
            <Select
              items={[
                { value: 'open', label: 'Auf Ziel — offener Posten' },
                { value: 'paid', label: 'Sofort bezahlt' },
              ]}
              value={settlement}
              onValueChange={(next) => setSettlement(next as Settlement)}
            />
          </Field>
          {settlement === 'paid' && (
            <Field label="Zahlungsmittel" hint="Bar heißt Kasse, nicht Bank">
              <Select
                items={paymentAccounts.map((a) => ({ value: a.number, label: `${a.number} ${a.name}` }))}
                value={paymentAccount}
                onValueChange={setPaymentAccount}
              />
            </Field>
          )}
        </div>

        {needsEntertainment && (
          <div className={cn(NOTE, NOTE_TONE.attention)}>
            <h4 className="text-label text-attention-text">
              Aufzeichnung zur Bewirtung
              <HelpPopover label="Erklärung zur Bewirtungsaufzeichnung">
                § 4 Abs. 5 Satz 1 Nr. 2 EStG verlangt Ort, Tag, Teilnehmer und Anlass. Ohne diese
                Angaben ist der Abzug auch für die abziehbaren 70 % verloren.
              </HelpPopover>
            </h4>
            <div className="mt-3 flex flex-col gap-3">
              <div className="grid grid-cols-2 gap-3">
                <Field label="Ort">
                  <Input
                    value={entertainment.place}
                    onChange={(e) => setEntertainment({ ...entertainment, place: e.target.value })}
                  />
                </Field>
                <Field label="Tag">
                  <Input
                    type="date"
                    value={entertainment.day}
                    onChange={(e) => setEntertainment({ ...entertainment, day: e.target.value })}
                  />
                </Field>
              </div>
              <Field label="Teilnehmer" hint="alle bewirteten Personen, mit Firma">
                <Input
                  value={entertainment.participants}
                  onChange={(e) => setEntertainment({ ...entertainment, participants: e.target.value })}
                />
              </Field>
              <Field label="Anlass" hint="der konkrete geschäftliche Anlass">
                <Input
                  value={entertainment.occasion}
                  onChange={(e) => setEntertainment({ ...entertainment, occasion: e.target.value })}
                />
              </Field>
            </div>
          </div>
        )}

        <Field label="Buchungstext" hint="leer lassen für den Standardtext" optional>
          <Input value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
      </div>

      <div className="mt-6 flex flex-col gap-3">
        {proposal && <ProposalNotes proposal={proposal} />}
        {hasUnreadableAmount && (
          <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>
            Ein Nettobetrag ist nicht lesbar. Erwartet wird etwa <span className="code-num">1234,56</span>{' '}
            — ohne Tausenderpunkt. Solange er so dasteht, fiele die Position aus der Buchung.
          </p>
        )}
        <PostingWarnings warnings={preview?.warnings} />
        <PostingPreviewPanel preview={preview} error={previewError} />
        {error && (
          <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>{error}</p>
        )}
      </div>

      <div className="mt-6 flex justify-end">
        <Button
          type="submit"
          variant="primary"
          loading={busy}
          disabled={hasUnreadableAmount || !preview?.balanced}
        >
          Buchen
        </Button>
      </div>
    </form>
  );
};

// -------------------------------------------------------------------------

/**
 * Das Prüfergebnis am Beleg.
 *
 * Der Prüfumfang wird benannt, nicht behauptet: die Referenzumsetzung von
 * EN 16931 ist ein Schematron-Regelwerk, das kein Go-Prozessor ausführt. Was
 * Buchfink prüft, ist eine belegte Teilmenge — das steht hier, statt
 * Vollständigkeit vorzutäuschen.
 */
const ValidationPanel: React.FC<{ receipt: Receipt }> = ({ receipt }) => {
  if (!receipt.validatedAt) return null;

  let findings: ValidationFinding[] = [];
  let unreadable = false;
  try {
    const parsed = receipt.validationFindings ? JSON.parse(receipt.validationFindings) : [];
    findings = Array.isArray(parsed) ? parsed : [];
    unreadable = !Array.isArray(parsed);
  } catch {
    unreadable = true;
  }
  const errors = findings.filter((f) => f.severity === 'fatal');
  const rest = findings.filter((f) => f.severity !== 'fatal');
  // Grün nur, wenn beide Quellen es sagen. Die Zahl kommt aus dem Backend und
  // überlebt jede Formatänderung an der Befundliste; ohne sie hieße eine Liste,
  // die sich nicht lesen lässt, „keine Verstöße" — die Falschaussage, die von
  // allen die teuerste ist.
  const clean = errors.length === 0 && !receipt.validationErrors && !unreadable;
  const complete = receipt.validationCoverage === 'full';

  return (
    <div className={cn(NOTE, clean ? NOTE_TONE.positive : NOTE_TONE.negative, 'mt-3')}>
      <h3
        className={cn(
          'flex items-start gap-2 text-label',
          clean ? 'text-positive-text' : 'text-negative-text',
        )}
      >
        {clean ? (
          <ShieldCheck className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        ) : (
          <AlertTriangle className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        )}
        {clean
          ? complete
            ? 'Alle Regeln der Norm erfüllt'
            : 'Geprüfte Regeln erfüllt'
          : errors.length > 0
            ? `${errors.length} ${errors.length === 1 ? 'Verstoß' : 'Verstöße'} gegen die geprüften Regeln`
            : `${receipt.validationErrors} ${
                receipt.validationErrors === 1 ? 'Verstoß' : 'Verstöße'
              } gegen die geprüften Regeln — die Einzelheiten sind nicht lesbar`}
      </h3>
      <p className="text-caption text-ink-muted mt-1">
        {receipt.detectedProfile} · Regelwerk {receipt.validationRuleset}
        {complete
          ? ' · alle 223 Geschäftsregeln der Norm'
          : ' · Teilprüfung, die Extension-Regeln bleiben offen'}
      </p>
      {[...errors, ...rest].map((f, i) => (
        <p key={i} className="text-caption text-ink-muted mt-1">
          <span className="code-num text-ink-subtle">{f.rule}</span>{' '}
          {f.where ? <span className="text-ink-subtle">{f.where}: </span> : null}
          {f.message}
        </p>
      ))}
    </div>
  );
};

/**
 * Warum aus dem strukturierten Teil kein Buchungsvorschlag wurde.
 *
 * Der Beleg bleibt buchbar — von Hand. Was das Backend ablehnt, ist der
 * Vorschlag, nicht die Buchung: eine Gutschrift ist ein gültiges Dokument mit
 * einem anderen Geschäftsvorfall dahinter, und diese Entscheidung trifft der
 * Nutzer, nicht Buchfink.
 */
const ProposalRefusal: React.FC<{ message: string | null }> = ({ message }) => {
  if (!message) return null;
  return (
    <div className={cn(NOTE, NOTE_TONE.attention, 'mb-5')}>
      <h3 className="flex items-start gap-2 text-label text-attention-text">
        <AlertTriangle className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        Kein Buchungsvorschlag aus dem Rechnungsdatensatz
      </h3>
      <p className="text-body text-ink-muted mt-1.5">{message}</p>
      <p className="text-caption text-ink-subtle mt-1">
        Der Beleg lässt sich weiterhin von Hand kontieren.
      </p>
    </div>
  );
};

/** Was aus dem strukturierten Teil kam und was offen blieb. */
const ProposalNotes: React.FC<{ proposal: EInvoiceProposal }> = ({ proposal }) => (
  <div className={cn(NOTE, NOTE_TONE.neutral)}>
    <h3 className="flex items-start gap-2 text-label text-ink">
      <FileCode className="w-4 h-4 mt-px shrink-0 text-ink-subtle" strokeWidth={1.5} />
      Aus der E-Rechnung übernommen
    </h3>
    <p className="text-caption text-ink-muted mt-1.5">
      {proposal.supplierName}
      {proposal.invoiceNumber && ` · Rechnung ${proposal.invoiceNumber}`}
      {proposal.kindLabel && ` · ${proposal.kindLabel}`}
      {proposal.profile && ` · Profil ${proposal.profile}`}
    </p>
    {proposal.precedingInvoices && proposal.precedingInvoices.length > 0 && (
      <p className="text-caption text-ink-muted mt-1">
        Bezug auf {proposal.precedingInvoices.join(', ')} — die Verrechnung mit der genannten
        Rechnung führt Buchfink noch nicht; sie ist von Hand zu prüfen.
      </p>
    )}
    {proposal.notes && proposal.notes.length > 0 && (
      <ul className="text-caption text-ink-muted list-disc pl-4 mt-1 space-y-0.5">
        {proposal.notes.map((note, i) => (
          <li key={i}>{note}</li>
        ))}
      </ul>
    )}
  </div>
);

/**
 * Hinweise zur Buchung.
 *
 * Sie blockieren nie. Was aus einer fehlenden E-Rechnung folgt, ist eine
 * Rechtsfrage — Buchfink zeigt sie an und bewertet sie nicht.
 */
const PostingWarnings: React.FC<{ warnings?: PostingWarning[] }> = ({ warnings }) => {
  const [copied, setCopied] = useState<string | null>(null);
  if (!warnings || warnings.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {warnings.map((warning) => {
        const loud = warning.severity === 'warning';
        return (
          <div key={warning.code} className={cn(NOTE, loud ? NOTE_TONE.attention : NOTE_TONE.neutral)}>
            <h3
              className={cn(
                'flex items-start gap-2 text-label',
                loud ? 'text-attention-text' : 'text-ink',
              )}
            >
              <AlertTriangle
                className={cn('w-4 h-4 mt-px shrink-0', !loud && 'text-ink-subtle')}
                strokeWidth={1.5}
              />
              {warning.title}
            </h3>
            <p className="text-body text-ink-muted mt-1.5">{warning.detail}</p>
            {warning.supplierNote && (
              <Button
                variant="quiet"
                size="sm"
                className="mt-2 -ml-2.5"
                onClick={() => {
                  void navigator.clipboard?.writeText(warning.supplierNote ?? '');
                  setCopied(warning.code);
                  window.setTimeout(() => setCopied(null), 2000);
                }}
              >
                {copied === warning.code ? 'Kopiert' : 'Hinweistext für den Lieferanten kopieren'}
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
};

/** Zeigt den Buchungssatz, den das Backend berechnet hat. */
const PostingPreviewPanel: React.FC<{ preview: PostingPreview | null; error: string | null }> = ({
  preview,
  error,
}) => {
  if (error) {
    return <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>{error}</p>;
  }
  if (!preview) {
    return (
      <p className="text-caption text-ink-subtle">
        Der Buchungssatz erscheint, sobald Lieferant und Positionen vollständig sind.
      </p>
    );
  }

  return (
    <Section title="Buchungssatz" divider={false} className="mt-2">
      <Table density="kompakt">
        <Tbody>
          {preview.lines.map((line, index) => (
            <Tr key={index}>
              <Td className="w-14 text-ink-subtle">{line.side === 'S' ? 'Soll' : 'Haben'}</Td>
              <Td code className="w-20">
                {line.account}
              </Td>
              <Td className="max-w-[16rem] truncate text-ink-muted">{line.accountName}</Td>
              <Td numeric>{formatCents(line.amount)}</Td>
            </Tr>
          ))}
          <Tr>
            <Td colSpan={3} className="text-ink-subtle">
              Netto
            </Td>
            <Td numeric className="text-ink-muted">
              {formatCents(preview.net)}
            </Td>
          </Tr>
          <Tr>
            <Td colSpan={3} className="text-ink-subtle">
              Steuer
            </Td>
            <Td numeric className="text-ink-muted">
              {formatCents(preview.tax)}
            </Td>
          </Tr>
          <Tr variant="sum">
            <Td colSpan={3}>Zahlbetrag</Td>
            <Td numeric>{formatCents(preview.gross)}</Td>
          </Tr>
        </Tbody>
      </Table>
    </Section>
  );
};
