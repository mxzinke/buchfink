import React, { useEffect, useMemo, useState } from 'react';
import { AlertCircle, Code, FileText, Plus, Trash2, Undo2 } from 'lucide-react';
import type {
  Contact,
  Invoice,
  ReceiptPreview,
  InvoiceItem,
  PostingPreview,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
} from '../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, formatTaxRate, parseCents } from '../utils/formatters';
import {
  Button,
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
  Th,
  Thead,
  Tr,
  cn,
  toast,
  type Status,
} from '../components/ui';

/**
 * Ausgangsrechnungen.
 *
 * Rechnungsnummer und Buchung entstehen beim Ausstellen im Backend: die Nummer
 * kommt aus dem lückenlosen Nummernkreis nach § 14 Abs. 4 Nr. 4 UStG, die
 * Forderung wird im selben Schritt gebucht. Es gibt keinen Zustand, in dem eine
 * Rechnung existiert, aber nicht im Journal steht.
 */

const STATUS: Record<Invoice['status'], Status> = {
  draft: 'entwurf',
  issued: 'offen',
  paid: 'ausgeglichen',
  cancelled: 'storniert',
};

interface DraftItem {
  description: string;
  quantity: string;
  unit: string;
  unitPrice: string;
  taxRate: TaxRate;
}

const newItem = (rate: TaxRate): DraftItem => ({
  description: '',
  quantity: '1',
  unit: 'Stück',
  unitPrice: '',
  taxRate: rate,
});

export const InvoicesPage: React.FC = () => {
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [preview, setPreview] = useState<{ invoice: Invoice; xml: string } | null>(null);
  const [documentPreview, setDocumentPreview] = useState<
    ({ invoice: Invoice } & ReceiptPreview) | null
  >(null);
  const [cancelling, setCancelling] = useState<Invoice | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [list, contactList, treatmentList] = await Promise.all([
        Api.getInvoices(),
        Api.getContacts(),
        Api.getTaxTreatments('outgoing'),
      ]);
      setInvoices(list);
      setContacts(contactList.filter((c) => c.type === 'customer'));
      setTreatments(treatmentList);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function showZugferd(invoice: Invoice) {
    try {
      const [xml] = await Api.generateInvoiceZUGFeRD(invoice.id);
      setPreview({ invoice, xml });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  /**
   * Öffnet das archivierte Rechnungsdokument.
   *
   * Gezeigt wird das hybride PDF, das beim Ausstellen abgelegt wurde — nicht eine
   * frische Darstellung. Was der Kunde bekommen hat, ist das, was hier erscheint.
   */
  async function showDocument(invoice: Invoice) {
    try {
      const doc = await Api.getInvoiceDocument(invoice.id);
      setDocumentPreview({ invoice, ...doc });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  const open = invoices.filter((i) => i.status === 'issued');
  const openTotal = open.reduce((sum, i) => sum + i.grossAmount, 0);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Ausgangsrechnungen"
        context={
          loading
            ? undefined
            : `${invoices.length} im Geschäftsjahr · ${open.length} offen über ${formatCents(openTotal)}`
        }
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zum Ausstellen">
              Ausstellen und Buchen sind ein Schritt. Die Rechnungsnummer wird lückenlos und
              fortlaufend vergeben, die Forderung sofort auf das Personenkonto des Kunden gebucht.
              Eine Rechnung, die nicht im Journal steht, kann es deshalb nicht geben.
            </HelpPopover>
            <Button
              variant="primary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              disabled={contacts.length === 0}
              title={contacts.length === 0 ? 'Zuerst einen Kunden in den Stammdaten anlegen' : undefined}
              onClick={() => setShowForm(true)}
            >
              Neue Rechnung
            </Button>
          </div>
        }
      />

      <Section divider={false} className="mt-8">
        {loading ? (
          <SkeletonRows rows={6} />
        ) : invoices.length === 0 ? (
          <EmptyState
            icon={<FileText className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch keine Rechnungen im aktiven Geschäftsjahr"
            description={
              contacts.length === 0
                ? 'Lege zuerst einen Kunden in den Stammdaten an.'
                : 'Die erste Rechnung wird beim Ausstellen sofort gebucht.'
            }
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-32">Nummer</Th>
                <Th className="w-28">Datum</Th>
                <Th>Kunde</Th>
                <Th numeric className="w-36">
                  Netto
                </Th>
                <Th numeric className="w-36">
                  Brutto
                </Th>
                <Th className="w-36">Status</Th>
                <Th className="w-28" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {invoices.map((invoice) => (
                <Tr
                  key={invoice.id}
                  variant={invoice.status === 'cancelled' ? 'storno' : 'default'}
                  className="group"
                >
                  <Td code>{invoice.invoiceNumber}</Td>
                  <Td className="text-ink-subtle num">{formatDate(invoice.date)}</Td>
                  <Td className="max-w-[20rem] truncate">{invoice.contactName}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(invoice.netAmount, invoice.currency)}
                  </Td>
                  <Td numeric>{formatCents(invoice.grossAmount, invoice.currency)}</Td>
                  <Td>
                    <StatusBadge status={STATUS[invoice.status]} />
                  </Td>
                  <Td className="pl-0">
                    <span
                      className="flex items-center justify-end gap-0.5 opacity-0 transition-opacity
                                 duration-120 ease-quiet group-hover:opacity-100 focus-within:opacity-100"
                    >
                      {invoice.receiptId && (
                        <Button
                          variant="quiet"
                          size="sm"
                          iconOnly
                          title="Rechnungsdokument ansehen"
                          aria-label={`Dokument zu ${invoice.invoiceNumber} ansehen`}
                          onClick={() => showDocument(invoice)}
                        >
                          <FileText className="w-4 h-4" strokeWidth={1.5} />
                        </Button>
                      )}
                      <Button
                        variant="quiet"
                        size="sm"
                        iconOnly
                        title="ZUGFeRD-XML ansehen"
                        aria-label={`XML zu ${invoice.invoiceNumber} ansehen`}
                        onClick={() => showZugferd(invoice)}
                      >
                        <Code className="w-4 h-4" strokeWidth={1.5} />
                      </Button>
                      {invoice.status === 'issued' && (
                        <Button
                          variant="quiet"
                          size="sm"
                          iconOnly
                          title="Rechnung stornieren"
                          aria-label={`Rechnung ${invoice.invoiceNumber} stornieren`}
                          onClick={() => setCancelling(invoice)}
                        >
                          <Undo2 className="w-4 h-4" strokeWidth={1.5} />
                        </Button>
                      )}
                    </span>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {showForm && (
        <InvoiceForm
          contacts={contacts}
          treatments={treatments}
          onClose={() => setShowForm(false)}
          onIssued={async (number) => {
            setShowForm(false);
            toast.success(`Rechnung ${number} ausgestellt und gebucht.`);
            await load();
          }}
        />
      )}

      <Dialog
        open={documentPreview !== null}
        onOpenChange={(next) => !next && setDocumentPreview(null)}
        title={documentPreview?.invoice.invoiceNumber ?? ''}
        width="max-w-4xl"
      >
        {documentPreview && (
          <>
            <p className="text-caption text-ink-subtle -mt-1 mb-3">
              Hybrides PDF/A-3 mit eingebettetem ZUGFeRD-XML — das archivierte Dokument, nicht eine
              neue Darstellung.
            </p>
            {!documentPreview.intact && (
              <div className="mb-3 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
                <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
                <p className="text-body text-negative-text">
                  Die Datei auf der Platte passt nicht mehr zu ihrer Prüfsumme.
                </p>
              </div>
            )}
            {/* Belegvorschau: fremdes Dokument, deshalb eigene Fläche (§6.2, Fall 3). */}
            <iframe
              title={documentPreview.fileName}
              src={documentPreview.dataUrl}
              className="w-full h-[62vh] rounded-card border border-line bg-sunken"
            />
          </>
        )}
      </Dialog>

      <Dialog
        open={preview !== null}
        onOpenChange={(next) => !next && setPreview(null)}
        title={`ZUGFeRD-XML · ${preview?.invoice.invoiceNumber ?? ''}`}
        width="max-w-3xl"
      >
        {/* Technische Ausgabe, deshalb Monospace auf dunkler Fläche (§4.1). */}
        <pre className="rounded-card border border-shell-line bg-shell text-shell-text
                        font-mono text-caption leading-relaxed p-5 max-h-[60vh] overflow-auto">
          {preview?.xml}
        </pre>
      </Dialog>

      <CancelDialog
        invoice={cancelling}
        onClose={() => setCancelling(null)}
        onDone={async () => {
          setCancelling(null);
          await load();
        }}
      />
    </div>
  );
};

// -------------------------------------------------------------------------

/** Kopf- und Zeilenraster der Positionen. Eine Definition für beide. */
const ITEM_GRID = 'grid grid-cols-[minmax(0,1fr)_5rem_6rem_7rem_6rem_7rem_2rem] gap-2 items-center';

const InvoiceForm: React.FC<{
  contacts: Contact[];
  treatments: TaxTreatmentInfo[];
  onClose: () => void;
  onIssued: (invoiceNumber: string) => void;
}> = ({ contacts, treatments, onClose, onIssued }) => {
  const today = new Date().toISOString().split('T')[0];
  const [contactId, setContactId] = useState(contacts[0]?.id ?? 0);
  const [date, setDate] = useState(today);
  const [serviceFrom, setServiceFrom] = useState(today);
  const [serviceTo, setServiceTo] = useState(today);
  const [treatment, setTreatment] = useState<TaxTreatment>('domestic');
  const [items, setItems] = useState<DraftItem[]>([newItem(TAX_RATE_STANDARD)]);
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<PostingPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const contact = contacts.find((c) => c.id === contactId);
  const treatmentInfo = treatments.find((t) => t.treatment === treatment);
  const taxable = treatment === 'domestic';

  // Die Positionen in der Form, die das Backend erwartet. Hier wird nur
  // umgerechnet, nicht gerechnet: Mengen von drei Nachkommastellen auf Milli,
  // Preise auf Cent.
  const draft = useMemo(
    () => ({
      contactId,
      date,
      serviceDateFrom: serviceFrom,
      serviceDateTo: serviceTo,
      taxTreatment: treatment,
      currency: 'EUR',
      items: items.map((item, index) => ({
        position: index + 1,
        description: item.description,
        quantityMilli: Math.round((Number(item.quantity.replace(',', '.')) || 0) * 1000),
        unit: item.unit,
        unitPrice: parseCents(item.unitPrice) ?? 0,
        taxRate: item.taxRate,
      })) as InvoiceItem[],
    }),
    [contactId, date, serviceFrom, serviceTo, treatment, items],
  );

  // Netto, Steuer und Brutto kommen aus dem Backend. Diese Maske hat die
  // Steuerrechnung früher selbst nachgebaut, samt Rundung je Steuersatzgruppe —
  // eine zweite Wahrheit, die auseinanderläuft, sobald ein Steuerfall dazukommt.
  useEffect(() => {
    const complete = contactId > 0 && draft.items.every((i) => i.description && i.unitPrice > 0);
    if (!complete) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewOutgoingInvoice(draft)
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
  }, [draft, contactId]);

  function update(index: number, patch: Partial<DraftItem>) {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  async function submit() {
    setBusy(true);
    try {
      const invoice = await Api.issueInvoice(draft);
      onIssued(invoice.invoiceNumber);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(next) => !next && onClose()}
      title="Neue Rechnung"
      width="max-w-4xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={!preview || preview.gross <= 0}
            onClick={submit}
          >
            Ausstellen und buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <Field
          label="Kunde"
          hint={
            contact
              ? `${contact.countryCode || 'DE'}${
                  contact.vatId ? ` · USt-IdNr. ${contact.vatId}` : ' · keine USt-IdNr. hinterlegt'
                }`
              : undefined
          }
        >
          <Select
            items={contacts.map((c) => ({ value: c.id, label: `${c.name} · Debitor ${c.ledgerAccount}` }))}
            value={contactId}
            onValueChange={setContactId}
          />
        </Field>

        <Field
          label="Steuerfall"
          hint={treatmentInfo?.hint}
          help="Der Steuerfall entscheidet über Erlöskonto und Steuerzeile. Für steuerfreie Lieferungen ins EU-Ausland ist die USt-IdNr. des Empfängers Voraussetzung."
        >
          <Select
            items={treatments.map((t) => ({ value: t.treatment, label: t.label }))}
            value={treatment}
            onValueChange={setTreatment}
          />
        </Field>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-4">
        <Field label="Rechnungsdatum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Leistung von">
          <Input type="date" value={serviceFrom} onChange={(e) => setServiceFrom(e.target.value)} />
        </Field>
        <Field label="Leistung bis">
          <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
        </Field>
      </div>

      <div className="mt-6 pt-6 border-t border-line">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-label text-ink-muted">Positionen</h3>
          <Button
            variant="quiet"
            size="sm"
            icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
            onClick={() => setItems((prev) => [...prev, newItem(TAX_RATE_STANDARD)])}
          >
            Position hinzufügen
          </Button>
        </div>

        <div className={cn(ITEM_GRID, 'text-caption text-ink-subtle mb-1')}>
          <span>Bezeichnung</span>
          <span className="text-right">Menge</span>
          <span>Einheit</span>
          <span className="text-right">Einzelpreis</span>
          <span>USt</span>
          <span className="text-right">Betrag</span>
          <span />
        </div>

        <div className="flex flex-col gap-2">
          {items.map((item, index) => (
            <div key={index} className={ITEM_GRID}>
              <Input
                value={item.description}
                onChange={(e) => update(index, { description: e.target.value })}
                aria-label={`Bezeichnung der Position ${index + 1}`}
              />
              <Input
                align="right"
                inputMode="decimal"
                value={item.quantity}
                onChange={(e) => update(index, { quantity: e.target.value })}
                aria-label={`Menge der Position ${index + 1}`}
              />
              <Input
                value={item.unit}
                onChange={(e) => update(index, { unit: e.target.value })}
                aria-label={`Einheit der Position ${index + 1}`}
              />
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={item.unitPrice}
                onChange={(e) => update(index, { unitPrice: e.target.value })}
                aria-label={`Einzelpreis der Position ${index + 1}`}
              />
              <Select
                items={[TAX_RATE_STANDARD, TAX_RATE_REDUCED, TAX_RATE_NONE].map((rate) => ({
                  value: rate,
                  label: formatTaxRate(rate),
                }))}
                value={item.taxRate}
                onValueChange={(taxRate) => update(index, { taxRate })}
                disabled={!taxable}
              />
              {/* Menge mal Einzelpreis — dieselbe Rechnung wie InvoiceItem.TotalNet
                  im Backend. Reine Darstellung: Steuer und Rundung je Satzgruppe
                  stehen weiter allein in der Buchungsvorschau. */}
              <span className="text-right num text-body text-ink">
                {formatCents(
                  Math.round(
                    ((draft.items[index]?.unitPrice ?? 0) * (draft.items[index]?.quantityMilli ?? 0)) /
                      1000,
                  ),
                )}
              </span>
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
        </div>

        <div className="mt-6 ml-auto w-72 text-body">
          {previewError ? (
            <p className="text-caption text-negative-text whitespace-pre-line">{previewError}</p>
          ) : !preview ? (
            <p className="text-caption text-ink-subtle text-right">
              Die Summen erscheinen, sobald Kunde und Positionen vollständig sind.
            </p>
          ) : (
            <>
              <div className="flex justify-between py-1">
                <span className="text-ink-muted">Nettobetrag</span>
                <span className="num">{formatCents(preview.net)}</span>
              </div>
              <div className="flex justify-between py-1">
                <span className="text-ink-muted">Umsatzsteuer</span>
                <span className="num">{formatCents(preview.tax)}</span>
              </div>
              <div className="flex justify-between pt-2 mt-1 rule-total font-semibold">
                <span>Gesamtbetrag</span>
                <span className="num">{formatCents(preview.gross)}</span>
              </div>
              {!taxable && (
                <p className="text-caption text-ink-subtle mt-2">
                  Steuerfreier Umsatz. Die Rechnung nennt den Befreiungsgrund.
                </p>
              )}
            </>
          )}
        </div>
      </div>
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const CancelDialog: React.FC<{
  invoice: Invoice | null;
  onClose: () => void;
  onDone: () => void;
}> = ({ invoice, onClose, onDone }) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (invoice) {
      setReason('');
      setError(null);
    }
  }, [invoice]);

  async function submit() {
    if (!reason.trim()) {
      setError('Ohne Grund lässt sich die Stornierung später nicht nachvollziehen.');
      return;
    }
    setBusy(true);
    try {
      await Api.cancelInvoice(invoice!.id, reason);
      toast.success(`Rechnung ${invoice!.invoiceNumber} storniert.`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={invoice !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Rechnung stornieren"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="danger" loading={busy} onClick={submit}>
            Stornieren
          </Button>
        </>
      }
    >
      {invoice && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{invoice.invoiceNumber}</span> über{' '}
            <span className="num text-ink">{formatCents(invoice.grossAmount, invoice.currency)}</span>{' '}
            an {invoice.contactName} wird per Generalumkehr zurückgenommen.
            <HelpPopover label="Erklärung zur Stornierung">
              Forderung, Erlös und Umsatzsteuer gehen auf null zurück. Die Rechnungsnummer bleibt
              vergeben und darf nicht neu verwendet werden — der Nummernkreis muss lückenlos bleiben.
            </HelpPopover>
          </p>

          <Field label="Grund der Stornierung" className="mt-4" error={error ?? undefined}>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Leistung nicht erbracht"
            />
          </Field>
        </>
      )}
    </Dialog>
  );
};
