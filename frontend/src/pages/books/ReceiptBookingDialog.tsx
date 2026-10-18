import React, { useEffect, useMemo, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import type {
  Account,
  Contact,
  JournalEntry,
  PostingGroup,
  PostingPreview,
  Receipt,
  ReceiptRequest,
  Settlement,
  TaxRate,
  TaxTreatment,
} from '../../types';
import { LIMIT_GIFT_PER_RECIPIENT, TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../../types';
import { Api } from '../../services/api';
import { usePostingLock } from '../../components/WriteLock';
import { formatCents, formatCentsPlain, parseCents } from '../../utils/formatters';
import {
  Button,
  DisabledReason,
  Combobox,
  Dialog,
  Field,
  FormGrid,
  Input,
  Notice,
  Select,
  SkeletonRows,
} from '../../components/ui';
import { DocumentPreview, PanelHeading, PostingLines, errorText } from './shared';

/** Die Kennung der Vorschau für einen fehlenden Leistungsnachweis. */
const SERVICE_PROOF_WARNING = 'service_proof_required';

const RATE_ITEMS: { value: TaxRate; label: string }[] = [
  { value: TAX_RATE_STANDARD, label: '19 %' },
  { value: TAX_RATE_REDUCED, label: '7 %' },
  { value: TAX_RATE_NONE, label: '0 %' },
];

interface Position {
  group: string;
  net: string;
  rate: TaxRate;
}

/**
 * Bewirtung und Geschenke verlangen eigene Aufzeichnungen (Teilnehmer,
 * Empfänger). Die Notfallmaske fragt sie nicht ab; solche Belege bucht Claude.
 */
const simpleGroup = (group: PostingGroup) =>
  !group.recipientRequired &&
  group.limit !== LIMIT_GIFT_PER_RECIPIENT &&
  group.deductibleQuota !== 'entertainment';

export interface ReceiptBookingDialogProps {
  receipt: Receipt;
  onClose: () => void;
  onBooked: (entry: JournalEntry) => Promise<void>;
}

/**
 * Die Notfallmaske für einen Eingangsbeleg: links der Beleg, rechts Lieferant,
 * Datum, Buchungsgruppe und Betrag. Den Buchungssatz rechnet das Backend; die Maske zeigt ihn vor
 * dem Buchen. Aus einer E-Rechnung kommen Lieferant, Datum und Beträge
 * vorbelegt, die Buchungsgruppe bleibt zu wählen.
 */
export const ReceiptBookingDialog: React.FC<ReceiptBookingDialogProps> = ({ receipt, onClose, onBooked }) => {
  const lock = usePostingLock();
  const [vendors, setVendors] = useState<Contact[] | null>(null);
  const [groups, setGroups] = useState<PostingGroup[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);

  const [contactId, setContactId] = useState<number | null>(null);
  const [documentDate, setDocumentDate] = useState(receipt.documentDate || receipt.receivedAt || '');
  const [description, setDescription] = useState(receipt.subject ?? '');
  const [positions, setPositions] = useState<Position[]>([{ group: '', net: '', rate: TAX_RATE_STANDARD }]);
  const [proposedTreatment, setProposedTreatment] = useState<TaxTreatment | ''>('');
  const [settlement, setSettlement] = useState<Settlement>('open');
  const [paymentAccount, setPaymentAccount] = useState('');
  const [overrideReason, setOverrideReason] = useState('');
  const [serviceProof, setServiceProof] = useState(receipt.serviceProof ?? '');

  const [preview, setPreview] = useState<PostingPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const hasStructured = receipt.files.some((f) => f.role === 'structured');
    Promise.all([
      Api.getSelectableContacts(),
      Api.getPostingGroups('incoming'),
      Api.getPaymentAccounts(),
      hasStructured ? Api.proposeFromEInvoice(receipt.id).catch(() => null) : Promise.resolve(null),
    ])
      .then(([contacts, groupList, accounts, proposal]) => {
        if (cancelled) return;
        const vendorList = (contacts ?? []).filter((c) => c.type === 'vendor');
        setVendors(vendorList);
        setGroups((groupList ?? []).filter(simpleGroup));
        setPaymentAccounts(accounts ?? []);
        setPaymentAccount(accounts?.[0]?.number ?? '');
        const byName = vendorList.find(
          (v) => receipt.issuerName && v.name.toLowerCase() === receipt.issuerName.toLowerCase(),
        );
        const request = proposal?.request;
        setContactId(request?.contactId || byName?.id || null);
        if (request) {
          if (request.documentDate) setDocumentDate(request.documentDate);
          if (request.description) setDescription(request.description);
          if (request.taxTreatment) setProposedTreatment(request.taxTreatment);
          if (request.positions.length > 0) {
            setPositions(
              request.positions.map((p) => ({ group: '', net: formatCentsPlain(p.net), rate: p.taxRate })),
            );
          }
        }
      })
      .catch((e) => !cancelled && setFailure(errorText(e)));
    return () => {
      cancelled = true;
    };
  }, [receipt]);

  const groupOf = (key: string) => groups.find((g) => g.key === key);
  const firstGroup = groupOf(positions[0]?.group ?? '');
  const treatment: TaxTreatment = proposedTreatment || firstGroup?.defaultTreatment || 'domestic';

  const request: ReceiptRequest | null = useMemo(() => {
    const parsed = positions.map((p) => ({ ...p, cents: parseCents(p.net) }));
    if (!contactId || parsed.some((p) => !p.group || p.cents === null || p.cents <= 0)) return null;
    return {
      contactId,
      receiptId: receipt.id,
      bookingDate: documentDate,
      documentDate,
      serviceDateFrom: documentDate,
      serviceDateTo: documentDate,
      description: description.trim() || receipt.subject || receipt.receiptNumber,
      taxTreatment: treatment,
      positions: parsed.map((p) => ({ postingGroup: p.group, net: p.cents ?? 0, taxRate: p.rate })),
      settlement,
      paymentAccount: settlement === 'paid' ? paymentAccount : undefined,
      currency: receipt.currency || 'EUR',
    };
  }, [contactId, positions, documentDate, description, treatment, settlement, paymentAccount, receipt]);

  // Den Buchungssatz rechnet das Backend; die Maske zeigt ihn nur.
  useEffect(() => {
    if (!request) {
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
          setPreviewError(errorText(e));
        });
    }, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [request]);

  const findings = preview?.inputTaxFindings ?? [];
  const proofRequired = (preview?.warnings ?? []).some((w) => w.code === SERVICE_PROOF_WARNING);
  const warnings = (preview?.warnings ?? []).filter((w) => w.code !== SERVICE_PROOF_WARNING);

  const blockedReason = !request
    ? 'Lieferant, Buchungsgruppe und Nettobetrag fehlen noch.'
    : !preview?.balanced
      ? 'Der Buchungssatz steht noch aus.'
      : findings.length > 0 && !overrideReason.trim()
        ? 'Ohne Grund wird eine Rechnung mit fehlender Pflichtangabe nicht mit Vorsteuer gebucht.'
        : proofRequired && !serviceProof.trim()
          ? 'Ab diesem Betrag braucht die Buchung einen Leistungsnachweis.'
          : lock.hint;

  function update(index: number, patch: Partial<Position>) {
    setPositions((prev) => prev.map((p, i) => (i === index ? { ...p, ...patch } : p)));
  }

  async function submit() {
    if (!request) return;
    setBusy(true);
    setFailure(null);
    try {
      const entry = await Api.postIncomingReceipt({
        ...request,
        overrideReason: overrideReason.trim() || undefined,
        serviceProof: serviceProof.trim() || undefined,
      });
      await onBooked(entry);
    } catch (e) {
      setFailure(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const groupItems = groups.map((g) => ({ value: g.key, label: g.label, meta: g.category }));
  const dirty = positions.some((p) => p.group !== '' || p.net !== '');

  return (
    <Dialog
      open
      onOpenChange={(next) => !next && onClose()}
      title={`Beleg ${receipt.receiptNumber} buchen`}
      width="max-w-6xl"
      dirty={dirty}
      footer={
        <>
          <DisabledReason id="beleg-buchen-grund" className="mr-auto self-center max-w-lg">
            {blockedReason}
          </DisabledReason>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={Boolean(blockedReason) || lock.locked}
            aria-describedby={blockedReason ? 'beleg-buchen-grund' : undefined}
            onClick={() => void submit()}
          >
            Buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] gap-8">
        <DocumentPreview
          className="lg:sticky lg:top-0 self-start"
          load={() => Api.getReceiptPreview(receipt.id)}
          loadKey={`${receipt.id}-${receipt.receiptHash}`}
        />
        {!vendors ? (
          <SkeletonRows rows={6} />
        ) : (
          <div>
            <FormGrid>
              <Field
                label="Lieferant"
                hint={vendors.length === 0 ? 'Erst unter Unternehmen → Kontakte anlegen' : undefined}
              >
                <Combobox<number>
                  items={vendors.map((v) => ({ value: v.id, label: v.name, meta: v.ledgerAccount }))}
                  value={contactId}
                  onValueChange={setContactId}
                  placeholder="Lieferant suchen"
                />
              </Field>
              <Field label="Belegdatum">
                <Input type="date" value={documentDate} onChange={(e) => setDocumentDate(e.target.value)} />
              </Field>
              <Field label="Buchungstext" className="md:col-span-2">
                <Input value={description} onChange={(e) => setDescription(e.target.value)} />
              </Field>
            </FormGrid>

            <div className="mt-6">
              <div className="grid grid-cols-[minmax(0,1fr)_8rem_6rem_2rem] gap-2 text-label text-ink-muted mb-1">
                <span>Buchungsgruppe</span>
                <span className="text-right">Netto</span>
                <span>Steuer</span>
                <span />
              </div>
              {positions.map((position, index) => (
                <div
                  key={index}
                  className="grid grid-cols-[minmax(0,1fr)_8rem_6rem_2rem] gap-2 items-center mb-2"
                >
                  <Combobox<string>
                    items={groupItems}
                    value={position.group || null}
                    onValueChange={(key) => {
                      const group = groupOf(key ?? '');
                      update(index, { group: key ?? '', rate: group?.defaultRate ?? position.rate });
                    }}
                    placeholder="Gruppe suchen"
                  />
                  <Input
                    align="right"
                    value={position.net}
                    onChange={(e) => update(index, { net: e.target.value })}
                    placeholder="0,00"
                    aria-label={`Nettobetrag Position ${index + 1}`}
                  />
                  <Select<TaxRate>
                    items={RATE_ITEMS}
                    value={position.rate}
                    onValueChange={(rate) => update(index, { rate })}
                    aria-label={`Steuersatz Position ${index + 1}`}
                  />
                  {positions.length > 1 ? (
                    <Button
                      variant="quiet"
                      size="sm"
                      iconOnly
                      title="Position entfernen"
                      aria-label={`Position ${index + 1} entfernen`}
                      onClick={() => setPositions((prev) => prev.filter((_, i) => i !== index))}
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </Button>
                  ) : (
                    <span />
                  )}
                </div>
              ))}
              <Button
                variant="quiet"
                size="sm"
                icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
                onClick={() =>
                  setPositions((prev) => [...prev, { group: '', net: '', rate: TAX_RATE_STANDARD }])
                }
              >
                Position
              </Button>
              <p className="mt-1 text-caption text-ink-subtle">
                Bewirtungen und Geschenke bucht Claude, weil dazu weitere Angaben gehören.
              </p>
            </div>

            <FormGrid className="mt-6">
              <Field label="Bezahlung">
                <Select<Settlement>
                  items={[
                    { value: 'open', label: 'Noch offen, Zahlung kommt über die Bank' },
                    { value: 'paid', label: 'Schon bezahlt' },
                  ]}
                  value={settlement}
                  onValueChange={setSettlement}
                />
              </Field>
              {settlement === 'paid' && (
                <Field label="Bezahlt über">
                  <Select<string>
                    items={paymentAccounts.map((a) => ({
                      value: a.number,
                      label: `${a.number} · ${a.name}`,
                    }))}
                    value={paymentAccount}
                    onValueChange={setPaymentAccount}
                  />
                </Field>
              )}
            </FormGrid>

            <div className="mt-6">
              <PanelHeading>Buchungssatz</PanelHeading>
              {previewError ? (
                <Notice tone="negative" text={previewError} />
              ) : preview ? (
                <PostingLines lines={preview.lines} currency={receipt.currency || 'EUR'} />
              ) : (
                <p className="text-caption text-ink-subtle">
                  Erscheint, sobald Lieferant, Buchungsgruppe und Nettobetrag stehen.
                </p>
              )}
              {preview && (
                <p className="mt-2 text-caption text-ink-subtle num">
                  {`Netto ${formatCents(preview.net)} · Steuer ${formatCents(preview.tax)} · Brutto ${formatCents(preview.gross)}`}
                </p>
              )}
            </div>

            {warnings.map((warning) => (
              <Notice key={warning.code} className="mt-4" text={`${warning.title}. ${warning.detail}`} />
            ))}

            {findings.length > 0 && (
              <div className="mt-4">
                <Notice
                  tone="negative"
                  text={`Der Rechnung fehlen Pflichtangaben: ${findings.map((f) => f.title).join(', ')}.`}
                />
                <Field label="Grund für das Buchen mit Vorsteuer" className="mt-3">
                  <Input value={overrideReason} onChange={(e) => setOverrideReason(e.target.value)} />
                </Field>
              </div>
            )}

            {proofRequired && (
              <Field
                label="Leistungsnachweis"
                hint="Pflichtangabe: Wer hat die Leistung geprüft?"
                className="mt-4"
              >
                <Input value={serviceProof} onChange={(e) => setServiceProof(e.target.value)} />
              </Field>
            )}

            {failure && <Notice tone="negative" className="mt-4" text={failure} />}
          </div>
        )}
      </div>
    </Dialog>
  );
};
