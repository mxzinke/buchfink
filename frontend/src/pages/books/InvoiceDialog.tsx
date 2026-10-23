import React, { useEffect, useMemo, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import type {
  Contact,
  Invoice,
  InvoiceItem,
  PostingPreview,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
  UnitCode,
} from '../../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../../types';
import { Api } from '../../services/api';
import { usePostingLock } from '../../components/WriteLock';
import { formatCents, parseCents } from '../../utils/formatters';
import {
  Button,
  DisabledReason,
  Combobox,
  ConfirmDialog,
  Dialog,
  Field,
  FormGrid,
  Input,
  Notice,
  Select,
  SkeletonRows,
} from '../../components/ui';
import { PanelHeading, PostingLines, errorText, todayISO } from './shared';

const RATE_ITEMS: { value: TaxRate; label: string }[] = [
  { value: TAX_RATE_STANDARD, label: '19 %' },
  { value: TAX_RATE_REDUCED, label: '7 %' },
  { value: TAX_RATE_NONE, label: '0 %' },
];

interface Item {
  description: string;
  quantity: string;
  unit: string;
  price: string;
  rate: TaxRate;
}

const emptyItem = (): Item => ({
  description: '',
  quantity: '1',
  unit: 'C62',
  price: '',
  rate: TAX_RATE_STANDARD,
});

/** Menge in Tausendstel; Komma und Punkt sind beide als Dezimalzeichen erlaubt. */
function quantityMilli(text: string): number | null {
  const value = Number(text.trim().replace(',', '.'));
  return Number.isFinite(value) && value > 0 ? Math.round(value * 1000) : null;
}

export interface InvoiceDialogProps {
  onClose: () => void;
  onIssued: (invoice: Invoice) => Promise<void>;
}

/**
 * Die Notfallmaske für eine Ausgangsrechnung an einen Kunden im Inland:
 * Kunde, Datum, Leistungszeitraum, Positionen, Zahlungsziel. Den Buchungssatz
 * und die Pflichtangaben prüft das Backend vor dem Ausstellen. Rechnungen mit
 * Verlagerung der Steuerschuld, Abschläge und Korrekturen schreibt Claude.
 */
export const InvoiceDialog: React.FC<InvoiceDialogProps> = ({ onClose, onIssued }) => {
  const lock = usePostingLock();
  const today = todayISO();
  const [customers, setCustomers] = useState<Contact[] | null>(null);
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [units, setUnits] = useState<UnitCode[]>([]);

  const [contactId, setContactId] = useState<number | null>(null);
  const [treatment, setTreatment] = useState<TaxTreatment>('domestic');
  const [date, setDate] = useState(today);
  const [serviceFrom, setServiceFrom] = useState(today);
  const [serviceTo, setServiceTo] = useState(today);
  const [items, setItems] = useState<Item[]>([emptyItem()]);
  const [dueDays, setDueDays] = useState('14');

  const [preview, setPreview] = useState<PostingPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([Api.getSelectableContacts(), Api.getTaxTreatments('outgoing'), Api.getUnitCodes()])
      .then(([contacts, treatmentList, unitList]) => {
        if (cancelled) return;
        setCustomers((contacts ?? []).filter((c) => c.type === 'customer'));
        // Ohne bestätigte USt-IdNr. des Kunden geht eine Rechnung ins Ausland
        // nicht hinaus; diese Fälle führt Claude mit der Abfrage zusammen.
        setTreatments((treatmentList ?? []).filter((t) => !t.requiresVatId));
        setUnits(unitList ?? []);
      })
      .catch((e) => !cancelled && setFailure(errorText(e)));
    return () => {
      cancelled = true;
    };
  }, []);

  const customer = customers?.find((c) => c.id === contactId);
  useEffect(() => {
    if (customer?.paymentTermsDays) setDueDays(String(customer.paymentTermsDays));
  }, [customer]);

  const draft: Partial<Invoice> | null = useMemo(() => {
    const parsed = items.map((item) => ({
      ...item,
      milli: quantityMilli(item.quantity),
      cents: parseCents(item.price),
    }));
    const complete =
      contactId !== null &&
      parsed.every(
        (item) => item.description.trim() && item.milli !== null && item.cents !== null && item.cents > 0,
      );
    if (!complete) return null;
    return {
      contactId: contactId ?? 0,
      date,
      serviceDateFrom: serviceFrom,
      serviceDateTo: serviceTo,
      taxTreatment: treatment,
      currency: 'EUR',
      smallAmount: false,
      terms: { dueDays: Number.parseInt(dueDays, 10) || 0, discountPermille: 0, discountDays: 0 },
      items: parsed.map(
        (item, index): InvoiceItem => ({
          position: index + 1,
          description: item.description.trim(),
          quantityMilli: item.milli ?? 0,
          unit: item.unit,
          unitPrice: item.cents ?? 0,
          taxRate: item.rate,
        }),
      ),
    };
  }, [contactId, date, serviceFrom, serviceTo, treatment, items, dueDays]);

  useEffect(() => {
    if (!draft) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewOutgoingInvoice(draft)
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
  }, [draft]);

  const findings = preview?.issuingFindings ?? [];
  const blockedReason = !draft
    ? 'Kunde, Beschreibung, Menge und Preis jeder Position fehlen noch.'
    : !preview
      ? 'Der Buchungssatz steht noch aus.'
      : findings.length > 0
        ? 'Der Rechnung fehlen Pflichtangaben.'
        : lock.hint;

  function update(index: number, patch: Partial<Item>) {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  async function issue() {
    if (!draft) return;
    setBusy(true);
    setFailure(null);
    try {
      const invoice = await Api.issueInvoice(draft);
      await onIssued(invoice);
    } catch (e) {
      setFailure(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const unitItems = (units.length > 0 ? units : [{ code: 'C62', label: 'Stück' }]).map((u) => ({
    value: u.code,
    label: u.label,
  }));
  const dirty = items.some((item) => item.description.trim() !== '' || item.price.trim() !== '');
  const grid = 'grid grid-cols-[minmax(0,1fr)_5rem_7rem_7rem_5.5rem_2rem] gap-2 items-center';

  return (
    <Dialog
      open
      onOpenChange={(next) => !next && onClose()}
      title="Rechnung schreiben"
      width="max-w-4xl"
      dirty={dirty}
      footer={
        <>
          <DisabledReason id="rechnung-grund" className="mr-auto self-center max-w-lg">
            {blockedReason}
          </DisabledReason>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={Boolean(blockedReason) || lock.locked}
            aria-describedby={blockedReason ? 'rechnung-grund' : undefined}
            onClick={() => setConfirming(true)}
          >
            Ausstellen
          </Button>
        </>
      }
    >
      {!customers ? (
        <SkeletonRows rows={6} />
      ) : (
        <>
          <FormGrid>
            <Field
              label="Kunde"
              hint={customers.length === 0 ? 'Erst unter Unternehmen → Kontakte anlegen' : undefined}
            >
              <Combobox<number>
                items={customers.map((c) => ({ value: c.id, label: c.name, meta: c.ledgerAccount }))}
                value={contactId}
                onValueChange={setContactId}
                placeholder="Kunde suchen"
              />
            </Field>
            <Field label="Steuerfall">
              <Select<TaxTreatment>
                items={treatments.map((t) => ({ value: t.treatment, label: t.label }))}
                value={treatment}
                onValueChange={setTreatment}
              />
            </Field>
            <Field label="Rechnungsdatum">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field label="Zahlungsziel in Tagen">
              <Input align="right" value={dueDays} onChange={(e) => setDueDays(e.target.value)} />
            </Field>
            <Field label="Leistung von">
              <Input type="date" value={serviceFrom} onChange={(e) => setServiceFrom(e.target.value)} />
            </Field>
            <Field label="Leistung bis">
              <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
            </Field>
          </FormGrid>

          <div className="mt-6">
            <div className={`${grid} text-label text-ink-muted mb-1`}>
              <span>Beschreibung</span>
              <span className="text-right">Menge</span>
              <span>Einheit</span>
              <span className="text-right">Einzelpreis netto</span>
              <span>Steuer</span>
              <span />
            </div>
            {items.map((item, index) => (
              <div key={index} className={`${grid} mb-2`}>
                <Input
                  value={item.description}
                  onChange={(e) => update(index, { description: e.target.value })}
                  aria-label={`Beschreibung Position ${index + 1}`}
                />
                <Input
                  align="right"
                  value={item.quantity}
                  onChange={(e) => update(index, { quantity: e.target.value })}
                  aria-label={`Menge Position ${index + 1}`}
                />
                <Select<string>
                  items={unitItems}
                  value={item.unit}
                  onValueChange={(unit) => update(index, { unit })}
                  aria-label={`Einheit Position ${index + 1}`}
                />
                <Input
                  align="right"
                  value={item.price}
                  placeholder="0,00"
                  onChange={(e) => update(index, { price: e.target.value })}
                  aria-label={`Einzelpreis Position ${index + 1}`}
                />
                <Select<TaxRate>
                  items={RATE_ITEMS}
                  value={item.rate}
                  onValueChange={(rate) => update(index, { rate })}
                  aria-label={`Steuersatz Position ${index + 1}`}
                />
                {items.length > 1 ? (
                  <Button
                    variant="quiet"
                    size="sm"
                    iconOnly
                    title="Position entfernen"
                    aria-label={`Position ${index + 1} entfernen`}
                    onClick={() => setItems((prev) => prev.filter((_, i) => i !== index))}
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
              onClick={() => setItems((prev) => [...prev, emptyItem()])}
            >
              Position
            </Button>
          </div>

          <div className="mt-6">
            <PanelHeading>Buchungssatz</PanelHeading>
            {previewError ? (
              <Notice tone="negative" text={previewError} />
            ) : preview ? (
              <>
                <PostingLines lines={preview.lines} />
                <p className="mt-2 text-caption text-ink-subtle num">
                  {`Netto ${formatCents(preview.net)} · Steuer ${formatCents(preview.tax)} · Brutto ${formatCents(preview.gross)}`}
                </p>
              </>
            ) : (
              <p className="text-caption text-ink-subtle">Erscheint, sobald Kunde und Positionen stehen.</p>
            )}
          </div>

          {findings.length > 0 && (
            <Notice className="mt-4" tone="negative" text={`Es fehlen Angaben: ${findings.join(' ')}`} />
          )}
          {failure && <Notice className="mt-4" tone="negative" text={failure} />}
        </>
      )}

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title="Rechnung ausstellen"
        description="Buchfink vergibt die nächste Rechnungsnummer und bucht die Forderung. Danach lässt sich die Rechnung nur noch mit einer Stornorechnung zurücknehmen."
        confirmLabel="Ausstellen"
        onConfirm={() => void issue()}
      />
    </Dialog>
  );
};
