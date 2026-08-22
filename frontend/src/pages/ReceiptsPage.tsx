import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import {
  AlertTriangle,
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
  PostingGroup,
  PostingPreview,
  Receipt,
  ReceiptFileInput,
  ReceiptFileRole,
  ReceiptRequest,
  Settlement,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
} from '../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, parseCents } from '../utils/formatters';
import { ErrorBox, Field, PrimaryButton, SecondaryButton, inputClass } from '../components/Form';
import { HelpTooltip } from '../components/HelpTooltip';

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

const STATUS_BADGES: Record<Receipt['status'], { label: string; classes: string }> = {
  filed: { label: 'Abgelegt', classes: 'bg-amber-100 text-amber-800' },
  sealed: { label: 'Gebucht', classes: 'bg-emerald-100 text-emerald-800' },
  discarded: { label: 'Verworfen', classes: 'bg-stone-100 text-stone-500' },
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
  const [loading, setLoading] = useState(true);
  const [filing, setFiling] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

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
      toast.success(`Beleg ${receipt.receiptNumber} abgelegt`);
      await load();
      setSelected(receipt);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setFiling(false);
    }
  }

  const vendors = useMemo(() => contacts.filter((c) => c.type === 'vendor'), [contacts]);

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Belege
            <HelpTooltip
              title="Ablegen und Buchen"
              content="Belege werden zuerst abgelegt und dann gebucht. Ein Beleg kann aus mehreren Dateien bestehen — eine ZUGFeRD-Rechnung ist ein PDF mit eingebettetem XML, eine XRechnung ist reines XML und braucht vor dem Buchen eine erzeugte Darstellung."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Eingangsbelege ablegen, ansehen und buchen
          </p>
        </div>
        <div className="flex gap-2">
          <SecondaryButton onClick={() => void load()} disabled={loading}>
            <RefreshCw className={`w-3.5 h-3.5 inline mr-1.5 ${loading ? 'animate-spin' : ''}`} />
            Aktualisieren
          </SecondaryButton>
          <PrimaryButton onClick={() => void fileReceipt()} disabled={filing}>
            <Plus className="w-3.5 h-3.5 inline mr-1.5" />
            Beleg ablegen
          </PrimaryButton>
        </div>
      </div>

      <ErrorBox message={error} />

      <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,22rem)_1fr] gap-5">
        <ReceiptList
          receipts={receipts}
          loading={loading}
          selectedId={selected?.id}
          onSelect={setSelected}
        />

        {selected ? (
          <ReceiptDetail
            key={selected.id}
            receipt={selected}
            vendors={vendors}
            groups={groups}
            treatments={treatments}
            paymentAccounts={paymentAccounts}
            onChanged={async (updated) => {
              setSelected(updated);
              await load();
            }}
            onBooked={async (entryNumber) => {
              toast.success(`Beleg gebucht als ${entryNumber}`);
              setSelected(null);
              await load();
            }}
          />
        ) : (
          <div className="rounded-2xl border border-dashed border-stone-200 bg-white/50 p-12 text-center text-xs text-stone-400">
            Wählen Sie links einen Beleg aus, um ihn anzusehen und zu buchen.
          </div>
        )}
      </div>
    </div>
  );
};

const ReceiptList: React.FC<{
  receipts: Receipt[];
  loading: boolean;
  selectedId?: number;
  onSelect: (r: Receipt) => void;
}> = ({ receipts, loading, selectedId, onSelect }) => {
  if (loading) {
    return (
      <div className="rounded-2xl border border-stone-200 bg-white p-8 text-center text-xs text-stone-400">
        Belege werden geladen…
      </div>
    );
  }
  if (receipts.length === 0) {
    return (
      <div className="rounded-2xl border border-stone-200 bg-white p-8 text-center text-xs text-stone-400">
        Noch keine Belege abgelegt.
      </div>
    );
  }

  return (
    <div className="rounded-2xl border border-stone-200 bg-white divide-y divide-stone-100 overflow-hidden">
      {receipts.map((receipt) => {
        const badge = STATUS_BADGES[receipt.status];
        const original = receipt.files.find((f) => f.role === 'original');
        return (
          <button
            key={receipt.id}
            onClick={() => onSelect(receipt)}
            className={`w-full text-left px-4 py-3 hover:bg-stone-50 transition-colors ${
              selectedId === receipt.id ? 'bg-amber-50/60' : ''
            }`}
          >
            <div className="flex items-center justify-between gap-2">
              <span className="font-mono text-xs font-semibold text-stone-800">
                {receipt.receiptNumber}
              </span>
              <span className={`px-1.5 py-0.5 rounded text-[10px] font-semibold ${badge.classes}`}>
                {badge.label}
              </span>
            </div>
            <div className="mt-1 text-[11px] text-stone-500 truncate">
              {original?.fileName ?? '—'}
            </div>
            <div className="mt-1 flex items-center gap-2 text-[10px] text-stone-400">
              <span>{receipt.receivedAt ? formatDate(receipt.receivedAt) : '—'}</span>
              {receipt.files.length > 1 && (
                <span className="inline-flex items-center gap-0.5">
                  <Paperclip className="w-3 h-3" />
                  {receipt.files.length}
                </span>
              )}
            </div>
          </button>
        );
      })}
    </div>
  );
};

const ReceiptDetail: React.FC<{
  receipt: Receipt;
  vendors: Contact[];
  groups: PostingGroup[];
  treatments: TaxTreatmentInfo[];
  paymentAccounts: Account[];
  onChanged: (updated: Receipt) => Promise<void>;
  onBooked: (entryNumber: string) => Promise<void>;
}> = ({ receipt, vendors, groups, treatments, paymentAccounts, onChanged, onBooked }) => (
  <div className="grid grid-cols-1 xl:grid-cols-2 gap-5 items-start">
    <ReceiptViewer receipt={receipt} onChanged={onChanged} />
    {receipt.status === 'filed' ? (
      <BookingForm
        receipt={receipt}
        vendors={vendors}
        groups={groups}
        treatments={treatments}
        paymentAccounts={paymentAccounts}
        onBooked={onBooked}
      />
    ) : (
      <div className="rounded-2xl border border-stone-200 bg-white p-5 text-xs text-stone-500 space-y-2">
        <div className="flex items-center gap-1.5 font-semibold text-stone-700">
          <ShieldCheck className="w-4 h-4 text-emerald-600" />
          {receipt.status === 'sealed' ? 'Gebucht und versiegelt' : 'Verworfen'}
        </div>
        {receipt.status === 'sealed' && (
          <p className="leading-relaxed">
            Der Beleg ist mit der Buchung versiegelt. Was später dazukommt — eine Mahnung, ein
            Zahlungsnachweis, eine korrigierte Rechnung — ist ein eigener Beleg auf denselben
            Geschäftsvorfall. Eine inhaltliche Korrektur läuft über Storno der Buchung.
          </p>
        )}
        {receipt.discardReason && <p>Grund: {receipt.discardReason}</p>}
      </div>
    )}
  </div>
);

const ReceiptViewer: React.FC<{
  receipt: Receipt;
  onChanged: (updated: Receipt) => Promise<void>;
}> = ({ receipt, onChanged }) => {
  const [preview, setPreview] = useState<{ dataUrl: string; mimeType: string; intact: boolean } | null>(
    null,
  );
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

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

  async function discard() {
    const reason = window.prompt('Warum wird der Beleg verworfen?');
    if (!reason) return;
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
    <div className="rounded-2xl border border-stone-200 bg-white overflow-hidden">
      <div className="px-4 py-3 border-b border-stone-100 flex items-center justify-between gap-2">
        <div>
          <div className="font-mono text-xs font-semibold text-stone-800">
            {receipt.receiptNumber}
          </div>
          <div className="text-[10px] text-stone-400 font-mono">
            Beleg-Hash {receipt.receiptHash.slice(0, 12)}…
          </div>
        </div>
        {open && (
          <SecondaryButton onClick={() => void discard()} disabled={busy}>
            Verwerfen
          </SecondaryButton>
        )}
      </div>

      <div className="bg-stone-50 border-b border-stone-100 min-h-[18rem] flex items-center justify-center p-3">
        {previewError ? (
          <div className="text-center space-y-2 max-w-sm">
            <AlertTriangle className="w-6 h-6 text-amber-500 mx-auto" />
            <p className="text-xs text-stone-600 leading-relaxed">{previewError}</p>
            {open && (
              <PrimaryButton onClick={() => void addFile('rendering')} disabled={busy}>
                Darstellung hinzufügen
              </PrimaryButton>
            )}
          </div>
        ) : !preview ? (
          <span className="text-xs text-stone-400">Vorschau wird geladen…</span>
        ) : preview.mimeType === 'application/pdf' ? (
          <iframe title="Belegvorschau" src={preview.dataUrl} className="w-full h-[28rem] rounded-lg bg-white" />
        ) : (
          <img src={preview.dataUrl} alt="Beleg" className="max-h-[28rem] rounded-lg" />
        )}
      </div>

      {preview && !preview.intact && (
        <div className="px-4 py-2 bg-rose-50 border-b border-rose-100 text-[11px] text-rose-700">
          Die Datei auf der Platte passt nicht mehr zu ihrer Prüfsumme. Sie wurde nach dem Ablegen
          verändert.
        </div>
      )}

      <ul className="divide-y divide-stone-100">
        {receipt.files.map((file) => (
          <li key={file.id} className="px-4 py-2 flex items-center gap-2 text-xs">
            <FileText className="w-3.5 h-3.5 text-stone-400 shrink-0" />
            <span className="truncate flex-1 text-stone-700">{file.fileName}</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-stone-100 text-stone-500 shrink-0">
              {ROLE_LABELS[file.role]}
              {file.derived && ' · erzeugt'}
            </span>
            {open && receipt.files.length > 1 && file.role !== 'original' && (
              <button
                onClick={() => void removeFile(file.id)}
                disabled={busy}
                className="text-stone-300 hover:text-rose-600 shrink-0"
                title="Datei entfernen"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            )}
          </li>
        ))}
      </ul>

      {open && (
        <div className="px-4 py-2 border-t border-stone-100 flex gap-2">
          <SecondaryButton onClick={() => void addFile('attachment')} disabled={busy}>
            Anhang hinzufügen
          </SecondaryButton>
          {!receipt.files.some((f) => f.role === 'rendering') && (
            <SecondaryButton onClick={() => void addFile('rendering')} disabled={busy}>
              Darstellung hinzufügen
            </SecondaryButton>
          )}
        </div>
      )}
    </div>
  );
};

const BookingForm: React.FC<{
  receipt: Receipt;
  vendors: Contact[];
  groups: PostingGroup[];
  treatments: TaxTreatmentInfo[];
  paymentAccounts: Account[];
  onBooked: (entryNumber: string) => Promise<void>;
}> = ({ receipt, vendors, groups, treatments, paymentAccounts, onBooked }) => {
  const today = receipt.receivedAt || new Date().toISOString().split('T')[0];
  const [contactId, setContactId] = useState(vendors[0]?.id ?? 0);
  const [documentDate, setDocumentDate] = useState(today);
  const [bookingDate, setBookingDate] = useState(today);
  const [serviceFrom, setServiceFrom] = useState(today);
  const [serviceTo, setServiceTo] = useState(today);
  const [treatment, setTreatment] = useState<TaxTreatment>('domestic');
  const [positions, setPositions] = useState<DraftPosition[]>([emptyPosition(groups[0])]);
  const [settlement, setSettlement] = useState<Settlement>('open');
  const [paymentAccount, setPaymentAccount] = useState(paymentAccounts[0]?.number ?? '');
  const [description, setDescription] = useState('');

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
      taxTreatment: treatment,
      positions: positions
        .filter((p) => p.postingGroup && parseCents(p.net) !== null)
        .map((p) => ({
          postingGroup: p.postingGroup,
          net: parseCents(p.net) ?? 0,
          taxRate: p.taxRate,
          text: p.text || undefined,
        })),
      settlement,
      paymentAccount: settlement === 'paid' ? paymentAccount : undefined,
      currency: 'EUR',
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
    ],
  );

  // Der Buchungssatz kommt aus dem Backend. Das Frontend rechnet ihn nicht nach:
  // eine zweite Steuerrechnung hier wäre eine zweite Wahrheit.
  useEffect(() => {
    if (!contactId || request.positions.length === 0) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewIncomingReceipt(request)
        .then((p) => {
          if (cancelled) return;
          setPreview(p);
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
  }, [request, contactId]);

  function updatePosition(index: number, patch: Partial<DraftPosition>) {
    setPositions((prev) => prev.map((p, i) => (i === index ? { ...p, ...patch } : p)));
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

  const groupsByCategory = useMemo(() => {
    const map = new Map<string, PostingGroup[]>();
    for (const g of groups) {
      const list = map.get(g.category) ?? [];
      list.push(g);
      map.set(g.category, list);
    }
    return [...map.entries()];
  }, [groups]);

  if (vendors.length === 0) {
    return (
      <div className="rounded-2xl border border-stone-200 bg-white p-5 text-xs text-stone-600 leading-relaxed">
        Für einen Eingangsbeleg braucht es einen Lieferanten. Legen Sie ihn unter „Kunden &
        Lieferanten“ an — er bekommt dabei sein Personenkonto aus dem Kreditoren-Nummernkreis.
      </div>
    );
  }

  return (
    <form onSubmit={submit} className="rounded-2xl border border-stone-200 bg-white">
      <div className="px-4 py-3 border-b border-stone-100 font-semibold text-stone-900 text-sm">
        Beleg buchen
      </div>

      <div className="p-4 space-y-3">
        <Field label="Lieferant">
          <select
            className={inputClass}
            value={contactId}
            onChange={(e) => setContactId(Number(e.target.value))}
            required
          >
            {vendors.map((v) => (
              <option key={v.id} value={v.id}>
                {v.name} · {v.ledgerAccount}
              </option>
            ))}
          </select>
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Belegdatum" hint="Rechnungsdatum">
            <input
              type="date"
              className={inputClass}
              value={documentDate}
              onChange={(e) => setDocumentDate(e.target.value)}
              required
            />
          </Field>
          <Field label="Buchungsdatum" hint="bestimmt die Periode">
            <input
              type="date"
              className={inputClass}
              value={bookingDate}
              onChange={(e) => setBookingDate(e.target.value)}
              required
            />
          </Field>
          <Field label="Leistung von" hint="§ 14 Abs. 4 Nr. 6 UStG">
            <input
              type="date"
              className={inputClass}
              value={serviceFrom}
              onChange={(e) => setServiceFrom(e.target.value)}
              required
            />
          </Field>
          <Field label="Leistung bis">
            <input
              type="date"
              className={inputClass}
              value={serviceTo}
              onChange={(e) => setServiceTo(e.target.value)}
              required
            />
          </Field>
        </div>

        <Field
          label="Steuerfall"
          hint={treatments.find((t) => t.treatment === treatment)?.hint}
        >
          <select
            className={inputClass}
            value={treatment}
            onChange={(e) => setTreatment(e.target.value as TaxTreatment)}
          >
            {treatments.map((t) => (
              <option key={t.treatment} value={t.treatment}>
                {t.label}
              </option>
            ))}
          </select>
        </Field>

        <div className="space-y-2">
          <div className="text-xs font-semibold text-stone-600">Positionen</div>
          {positions.map((position, index) => (
            <div key={index} className="grid grid-cols-[1fr_6rem_5.5rem_auto] gap-2 items-start">
              <select
                className={inputClass}
                value={position.postingGroup}
                onChange={(e) => selectGroup(index, e.target.value)}
                required
              >
                <option value="">Gruppe wählen…</option>
                {groupsByCategory.map(([category, list]) => (
                  <optgroup key={category} label={category}>
                    {list.map((g) => (
                      <option key={g.key} value={g.key}>
                        {g.label}
                      </option>
                    ))}
                  </optgroup>
                ))}
              </select>
              <input
                className={inputClass}
                placeholder="Netto"
                inputMode="decimal"
                value={position.net}
                onChange={(e) => updatePosition(index, { net: e.target.value })}
                required
              />
              <select
                className={inputClass}
                value={position.taxRate}
                onChange={(e) => updatePosition(index, { taxRate: Number(e.target.value) })}
              >
                <option value={TAX_RATE_STANDARD}>19 %</option>
                <option value={TAX_RATE_REDUCED}>7 %</option>
                <option value={TAX_RATE_NONE}>ohne</option>
              </select>
              <button
                type="button"
                onClick={() => setPositions((prev) => prev.filter((_, i) => i !== index))}
                disabled={positions.length === 1}
                className="mt-1.5 text-stone-300 hover:text-rose-600 disabled:opacity-30"
                title="Position entfernen"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          ))}
          <SecondaryButton
            type="button"
            onClick={() => setPositions((prev) => [...prev, emptyPosition(groups[0])])}
          >
            <Plus className="w-3.5 h-3.5 inline mr-1" />
            Position
          </SecondaryButton>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Zahlung">
            <select
              className={inputClass}
              value={settlement}
              onChange={(e) => setSettlement(e.target.value as Settlement)}
            >
              <option value="open">Auf Ziel — offener Posten</option>
              <option value="paid">Sofort bezahlt</option>
            </select>
          </Field>
          {settlement === 'paid' && (
            <Field label="Zahlungsmittel" hint="Bar heißt Kasse, nicht Bank">
              <select
                className={inputClass}
                value={paymentAccount}
                onChange={(e) => setPaymentAccount(e.target.value)}
                required
              >
                {paymentAccounts.map((a) => (
                  <option key={a.number} value={a.number}>
                    {a.number} {a.name}
                  </option>
                ))}
              </select>
            </Field>
          )}
        </div>

        <Field label="Buchungstext" hint="leer lassen für den Standardtext">
          <input
            className={inputClass}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>

        <PostingPreviewPanel preview={preview} error={previewError} />
        <ErrorBox message={error} />
      </div>

      <div className="flex justify-end gap-2 px-4 py-3 border-t border-stone-100">
        <PrimaryButton type="submit" disabled={busy || !preview?.balanced}>
          {busy ? 'Wird gebucht…' : 'Buchen'}
        </PrimaryButton>
      </div>
    </form>
  );
};

/** Zeigt den Buchungssatz, den das Backend berechnet hat. */
const PostingPreviewPanel: React.FC<{ preview: PostingPreview | null; error: string | null }> = ({
  preview,
  error,
}) => {
  if (error) return <ErrorBox message={error} />;
  if (!preview) {
    return (
      <div className="rounded-lg border border-dashed border-stone-200 px-3 py-4 text-center text-[11px] text-stone-400">
        Der Buchungssatz erscheint, sobald Lieferant und Positionen vollständig sind.
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-stone-200 overflow-hidden">
      <table className="w-full text-xs">
        <tbody className="divide-y divide-stone-100">
          {preview.lines.map((line, index) => (
            <tr key={index}>
              <td className="px-2.5 py-1.5 w-12 text-stone-500">
                {line.side === 'S' ? 'Soll' : 'Haben'}
              </td>
              <td className="px-2.5 py-1.5 font-mono text-stone-700">{line.account}</td>
              <td className="px-2.5 py-1.5 text-stone-500 truncate">{line.accountName}</td>
              <td className="px-2.5 py-1.5 text-right font-mono text-stone-800">
                {formatCents(line.amount)}
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot className="bg-stone-50 text-[11px]">
          <tr>
            <td colSpan={3} className="px-2.5 py-1 text-stone-500">
              Netto
            </td>
            <td className="px-2.5 py-1 text-right font-mono">{formatCents(preview.net)}</td>
          </tr>
          <tr>
            <td colSpan={3} className="px-2.5 py-1 text-stone-500">
              Steuer
            </td>
            <td className="px-2.5 py-1 text-right font-mono">{formatCents(preview.tax)}</td>
          </tr>
          <tr className="font-semibold text-stone-800">
            <td colSpan={3} className="px-2.5 py-1">
              Zahlbetrag
            </td>
            <td className="px-2.5 py-1 text-right font-mono">{formatCents(preview.gross)}</td>
          </tr>
        </tfoot>
      </table>
    </div>
  );
};
