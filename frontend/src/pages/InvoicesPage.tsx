import React, { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AlertCircle, Code, FileText, Plus, RefreshCw, Trash2, Undo2, X } from 'lucide-react';
import type {
  Contact,
  Invoice,
  InvoiceItem,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
} from '../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, formatTaxRate, parseCents } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

/**
 * Ausgangsrechnungen.
 *
 * Rechnungsnummer und Buchung entstehen beim Ausstellen im Backend: die Nummer
 * kommt aus dem lückenlosen Nummernkreis nach § 14 Abs. 4 Nr. 4 UStG, die
 * Forderung wird im selben Schritt gebucht. Es gibt keinen Zustand, in dem eine
 * Rechnung existiert, aber nicht im Journal steht.
 */

const inputClass =
  'w-full px-2.5 py-1.5 text-sm rounded-lg border border-stone-200 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none';

const STATUS_LABELS: Record<Invoice['status'], { label: string; classes: string }> = {
  draft: { label: 'Entwurf', classes: 'bg-stone-100 text-stone-600' },
  issued: { label: 'Offen', classes: 'bg-amber-100 text-amber-800' },
  paid: { label: 'Bezahlt', classes: 'bg-emerald-100 text-emerald-800' },
  cancelled: { label: 'Storniert', classes: 'bg-rose-100 text-rose-700' },
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
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [preview, setPreview] = useState<{ invoice: Invoice; xml: string } | null>(null);
  const [cancelling, setCancelling] = useState<Invoice | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError(null);
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
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function showZugferd(invoice: Invoice) {
    setError(null);
    try {
      const [xml] = await Api.generateInvoiceZUGFeRD(invoice.id);
      setPreview({ invoice, xml });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="p-4 sm:p-6 space-y-5 max-w-6xl mx-auto">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold text-stone-900 tracking-tight">Ausgangsrechnungen</h1>
          <p className="text-sm text-stone-600">
            Ausstellen und Buchen sind ein Schritt. Die Rechnungsnummer wird lückenlos und fortlaufend
            vergeben, die Forderung sofort auf das Personenkonto des Kunden gebucht.
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          disabled={contacts.length === 0}
          className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg bg-amber-700 text-white hover:bg-amber-800 disabled:opacity-40"
          title={contacts.length === 0 ? 'Zuerst einen Kunden in den Stammdaten anlegen' : undefined}
        >
          <Plus className="w-4 h-4" />
          Neue Rechnung
        </button>
      </header>

      {error && (
        <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-xl p-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {loading ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-stone-400">
          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
          Rechnungen werden geladen…
        </div>
      ) : invoices.length === 0 ? (
        <div className="bg-white p-8 rounded-xl border border-stone-200/80 text-center text-sm text-stone-500">
          <FileText className="w-6 h-6 mx-auto mb-2 text-stone-300" />
          Noch keine Rechnungen im aktiven Geschäftsjahr.
          {contacts.length === 0 && ' Lege zuerst einen Kunden in den Stammdaten an.'}
        </div>
      ) : (
        <div className="bg-white rounded-2xl border border-stone-200/80 shadow-xs overflow-x-auto">
          <table className="w-full text-sm min-w-[44rem]">
            <thead>
              <tr className="text-[11px] uppercase tracking-wide text-stone-400 border-b border-stone-100">
                <th className="text-left font-medium px-4 py-2">Nummer</th>
                <th className="text-left font-medium px-2 py-2">Datum</th>
                <th className="text-left font-medium px-2 py-2">Kunde</th>
                <th className="text-right font-medium px-2 py-2">Netto</th>
                <th className="text-right font-medium px-2 py-2">Brutto</th>
                <th className="text-left font-medium px-2 py-2">Status</th>
                <th className="px-4 py-2" />
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-50">
              {invoices.map((invoice) => {
                const status = STATUS_LABELS[invoice.status];
                return (
                  <tr key={invoice.id} className="hover:bg-stone-50/60">
                    <td className="px-4 py-2 font-mono text-xs text-stone-700">{invoice.invoiceNumber}</td>
                    <td className="px-2 py-2 text-xs text-stone-600">{formatDate(invoice.date)}</td>
                    <td className="px-2 py-2 text-stone-900">{invoice.contactName}</td>
                    <td className="px-2 py-2 text-right font-mono text-xs tabular-nums text-stone-600">
                      {formatCents(invoice.netAmount, invoice.currency)}
                    </td>
                    <td className="px-2 py-2 text-right font-mono text-sm tabular-nums text-stone-900">
                      {formatCents(invoice.grossAmount, invoice.currency)}
                    </td>
                    <td className="px-2 py-2">
                      <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${status.classes}`}>
                        {status.label}
                      </span>
                    </td>
                    <td className="px-4 py-2 text-right whitespace-nowrap">
                      <button
                        onClick={() => showZugferd(invoice)}
                        title="ZUGFeRD-XML ansehen"
                        className="text-stone-400 hover:text-stone-800 p-1"
                      >
                        <Code className="w-4 h-4" />
                      </button>
                      {invoice.status === 'issued' && (
                        <button
                          onClick={() => setCancelling(invoice)}
                          title="Rechnung stornieren"
                          className="text-stone-400 hover:text-rose-600 p-1"
                        >
                          <Undo2 className="w-4 h-4" />
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {showForm && (
        <InvoiceForm
          contacts={contacts}
          treatments={treatments}
          onClose={() => setShowForm(false)}
          onIssued={async (number) => {
            setShowForm(false);
            toast.success(`Rechnung ${number} ausgestellt und gebucht`);
            await load();
          }}
        />
      )}

      {preview && (
        <div className="fixed inset-0 bg-stone-900/40 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-2xl border border-stone-200 shadow-xl w-full max-w-3xl max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between px-5 py-3 border-b border-stone-100">
              <h2 className="font-semibold text-stone-900">
                ZUGFeRD-XML · {preview.invoice.invoiceNumber}
              </h2>
              <button onClick={() => setPreview(null)} className="text-stone-400 hover:text-stone-700">
                <X className="w-5 h-5" />
              </button>
            </div>
            <pre className="p-5 overflow-auto text-[11px] font-mono text-stone-700 whitespace-pre-wrap">
              {preview.xml}
            </pre>
          </div>
        </div>
      )}

      {cancelling && (
        <CancelDialog
          invoice={cancelling}
          onClose={() => setCancelling(null)}
          onDone={async () => {
            setCancelling(null);
            await load();
          }}
        />
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

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
  const [error, setError] = useState<string | null>(null);

  const contact = contacts.find((c) => c.id === contactId);
  const treatmentInfo = treatments.find((t) => t.treatment === treatment);
  const taxable = treatment === 'domestic';

  const computed = useMemo(() => {
    const parsed = items.map((item) => {
      const quantityMilli = Math.round((Number(item.quantity.replace(',', '.')) || 0) * 1000);
      const unitPrice = parseCents(item.unitPrice) ?? 0;
      // Mengen mit drei Nachkommastellen, dann kaufmännisch auf den Cent.
      const net = Math.round((unitPrice * quantityMilli) / 1000);
      return { ...item, quantityMilli, unitPrice, net };
    });

    // Die Steuer wird je Steuersatzgruppe gerundet, nicht je Position — genau wie
    // im Backend, damit die Vorschau der gebuchten Rechnung entspricht.
    const byRate = new Map<TaxRate, number>();
    for (const item of parsed) byRate.set(item.taxRate, (byRate.get(item.taxRate) ?? 0) + item.net);

    let net = 0;
    let tax = 0;
    for (const [rate, groupNet] of byRate) {
      net += groupNet;
      if (taxable) tax += Math.round((groupNet * rate) / 10000);
    }
    return { parsed, net, tax, gross: net + tax };
  }, [items, taxable]);

  function update(index: number, patch: Partial<DraftItem>) {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const invoice = await Api.issueInvoice({
        contactId,
        date,
        serviceDateFrom: serviceFrom,
        serviceDateTo: serviceTo,
        taxTreatment: treatment,
        currency: 'EUR',
        items: computed.parsed.map((item, index) => ({
          position: index + 1,
          description: item.description,
          quantityMilli: item.quantityMilli,
          unit: item.unit,
          unitPrice: item.unitPrice,
          taxRate: item.taxRate,
        })) as InvoiceItem[],
      });
      onIssued(invoice.invoiceNumber);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-stone-900/40 flex items-start justify-center p-4 overflow-y-auto z-50">
      <form
        onSubmit={submit}
        className="bg-white rounded-2xl border border-stone-200 shadow-xl w-full max-w-3xl my-8"
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-stone-100">
          <h2 className="font-semibold text-stone-900">Neue Rechnung</h2>
          <button type="button" onClick={onClose} className="text-stone-400 hover:text-stone-700">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <label className="block">
              <span className="block text-xs font-medium text-stone-600 mb-1">Kunde</span>
              <select
                value={contactId}
                onChange={(e) => setContactId(Number(e.target.value))}
                className={inputClass}
                required
              >
                {contacts.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name} · Debitor {c.ledgerAccount}
                  </option>
                ))}
              </select>
              {contact && (
                <span className="block text-[11px] text-stone-500 mt-1">
                  {contact.countryCode || 'DE'}
                  {contact.vatId ? ` · USt-IdNr. ${contact.vatId}` : ' · keine USt-IdNr. hinterlegt'}
                </span>
              )}
            </label>

            <label className="block">
              <span className="block text-xs font-medium text-stone-600 mb-1">
                Steuerfall
                <HelpTooltip
                  title="Steuerfall"
                  content={
                    'Der Steuerfall entscheidet über Erlöskonto und Steuerzeile. Für steuerfreie ' +
                    'Lieferungen ins EU-Ausland ist die USt-IdNr. des Empfängers Voraussetzung.'
                  }
                />
              </span>
              <select
                value={treatment}
                onChange={(e) => setTreatment(e.target.value as TaxTreatment)}
                className={inputClass}
              >
                {treatments.map((t) => (
                  <option key={t.treatment} value={t.treatment}>
                    {t.label}
                  </option>
                ))}
              </select>
              {treatmentInfo && (
                <span className="block text-[11px] text-stone-500 mt-1">{treatmentInfo.hint}</span>
              )}
            </label>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <label className="block">
              <span className="block text-xs font-medium text-stone-600 mb-1">Rechnungsdatum</span>
              <input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                className={inputClass}
                required
              />
            </label>
            <label className="block">
              <span className="block text-xs font-medium text-stone-600 mb-1">Leistung von</span>
              <input
                type="date"
                value={serviceFrom}
                onChange={(e) => setServiceFrom(e.target.value)}
                className={inputClass}
                required
              />
            </label>
            <label className="block">
              <span className="block text-xs font-medium text-stone-600 mb-1">Leistung bis</span>
              <input
                type="date"
                value={serviceTo}
                onChange={(e) => setServiceTo(e.target.value)}
                className={inputClass}
                required
              />
            </label>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-stone-700 uppercase tracking-wide">
                Positionen
              </span>
              <button
                type="button"
                onClick={() => setItems((prev) => [...prev, newItem(TAX_RATE_STANDARD)])}
                className="text-xs text-amber-800 hover:text-amber-900 font-medium"
              >
                Position hinzufügen
              </button>
            </div>

            {items.map((item, index) => (
              <div key={index} className="flex flex-wrap gap-2 items-end">
                <label className="flex-1 min-w-[12rem]">
                  <span className="block text-[11px] text-stone-500 mb-1">Bezeichnung</span>
                  <input
                    value={item.description}
                    onChange={(e) => update(index, { description: e.target.value })}
                    className={inputClass}
                    required
                  />
                </label>
                <label className="w-20">
                  <span className="block text-[11px] text-stone-500 mb-1">Menge</span>
                  <input
                    value={item.quantity}
                    onChange={(e) => update(index, { quantity: e.target.value })}
                    inputMode="decimal"
                    className={`${inputClass} text-right font-mono`}
                    required
                  />
                </label>
                <label className="w-24">
                  <span className="block text-[11px] text-stone-500 mb-1">Einheit</span>
                  <input
                    value={item.unit}
                    onChange={(e) => update(index, { unit: e.target.value })}
                    className={inputClass}
                  />
                </label>
                <label className="w-28">
                  <span className="block text-[11px] text-stone-500 mb-1">Einzelpreis</span>
                  <input
                    value={item.unitPrice}
                    onChange={(e) => update(index, { unitPrice: e.target.value })}
                    placeholder="0,00"
                    inputMode="decimal"
                    className={`${inputClass} text-right font-mono`}
                    required
                  />
                </label>
                <label className="w-24">
                  <span className="block text-[11px] text-stone-500 mb-1">USt</span>
                  <select
                    value={item.taxRate}
                    onChange={(e) => update(index, { taxRate: Number(e.target.value) })}
                    className={inputClass}
                    disabled={!taxable}
                  >
                    <option value={TAX_RATE_STANDARD}>{formatTaxRate(TAX_RATE_STANDARD)}</option>
                    <option value={TAX_RATE_REDUCED}>{formatTaxRate(TAX_RATE_REDUCED)}</option>
                    <option value={TAX_RATE_NONE}>{formatTaxRate(TAX_RATE_NONE)}</option>
                  </select>
                </label>
                <div className="w-28 text-right pb-1.5 font-mono text-sm tabular-nums text-stone-800">
                  {formatCents(computed.parsed[index]?.net ?? 0)}
                </div>
                {items.length > 1 && (
                  <button
                    type="button"
                    onClick={() => setItems((prev) => prev.filter((_, i) => i !== index))}
                    className="text-stone-400 hover:text-rose-600 pb-2"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
              </div>
            ))}
          </div>

          <div className="bg-stone-50 border border-stone-200 rounded-lg px-4 py-3 space-y-1 text-sm">
            <div className="flex justify-between">
              <span className="text-stone-600">Nettobetrag</span>
              <span className="font-mono tabular-nums">{formatCents(computed.net)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-stone-600">Umsatzsteuer</span>
              <span className="font-mono tabular-nums">{formatCents(computed.tax)}</span>
            </div>
            <div className="flex justify-between font-semibold text-stone-900 pt-1 border-t border-stone-200">
              <span>Gesamtbetrag</span>
              <span className="font-mono tabular-nums">{formatCents(computed.gross)}</span>
            </div>
            {!taxable && (
              <p className="text-[11px] text-stone-500 pt-1">
                Steuerfreier Umsatz — die Rechnung weist keine Umsatzsteuer aus und nennt den
                Befreiungsgrund.
              </p>
            )}
          </div>

          {error && (
            <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-lg p-3">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 px-5 py-3 border-t border-stone-100">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-2 text-sm rounded-lg border border-stone-200 text-stone-700 hover:bg-stone-50"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            disabled={busy || computed.gross <= 0}
            className="px-3 py-2 text-sm rounded-lg bg-amber-700 text-white hover:bg-amber-800 disabled:opacity-40"
          >
            {busy ? 'Wird ausgestellt…' : 'Ausstellen und buchen'}
          </button>
        </div>
      </form>
    </div>
  );
};

// -------------------------------------------------------------------------

const CancelDialog: React.FC<{
  invoice: Invoice;
  onClose: () => void;
  onDone: () => void;
}> = ({ invoice, onClose, onDone }) => {
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await Api.cancelInvoice(invoice.id, reason);
      toast.success(`Rechnung ${invoice.invoiceNumber} storniert`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-stone-900/40 flex items-center justify-center p-4 z-50">
      <form onSubmit={submit} className="bg-white rounded-2xl border border-stone-200 shadow-xl w-full max-w-lg">
        <div className="px-5 py-3 border-b border-stone-100">
          <h2 className="font-semibold text-stone-900">Rechnung stornieren</h2>
        </div>
        <div className="p-5 space-y-3">
          <p className="text-sm text-stone-600">
            Rechnung <span className="font-mono">{invoice.invoiceNumber}</span> über{' '}
            <span className="font-mono">{formatCents(invoice.grossAmount)}</span> an {invoice.contactName}.
          </p>
          <p className="text-xs text-stone-500 bg-stone-50 border border-stone-200 rounded-lg p-3 leading-relaxed">
            Die Buchung wird per Generalumkehr zurückgenommen: Forderung, Erlös und Umsatzsteuer gehen auf
            null zurück. Die Rechnungsnummer bleibt vergeben — sie darf nicht neu verwendet werden.
          </p>
          <label className="block">
            <span className="block text-xs font-medium text-stone-600 mb-1">Grund</span>
            <input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="z. B. Leistung nicht erbracht"
              className={inputClass}
              required
            />
          </label>
          {error && (
            <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 text-sm rounded-lg p-3">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>
        <div className="flex justify-end gap-2 px-5 py-3 border-t border-stone-100">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-2 text-sm rounded-lg border border-stone-200 text-stone-700 hover:bg-stone-50"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            disabled={busy || !reason.trim()}
            className="px-3 py-2 text-sm rounded-lg bg-rose-700 text-white hover:bg-rose-800 disabled:opacity-40"
          >
            {busy ? 'Wird storniert…' : 'Stornieren'}
          </button>
        </div>
      </form>
    </div>
  );
};
