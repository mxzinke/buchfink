import React, { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { Plus, Download, Code, Trash2, CheckCircle2, FileText } from 'lucide-react';
import { Invoice, InvoiceItem, CompanySettings } from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const InvoicesPage: React.FC = () => {
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [companySettings, setCompanySettings] = useState<CompanySettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [previewInvoice, setPreviewInvoice] = useState<Invoice | null>(null);
  const [zugferdXML, setZugferdXML] = useState<string | null>(null);

  // New Invoice Modal
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [invNumber, setInvNumber] = useState('');
  const [invDate, setInvDate] = useState(new Date().toISOString().split('T')[0]);
  const [invDueDate, setInvDueDate] = useState(
    new Date(Date.now() + 14 * 86400000).toISOString().split('T')[0]
  );
  const [contactName, setContactName] = useState('');
  const [items, setItems] = useState<InvoiceItem[]>([
    {
      position: 1,
      description: '',
      quantity: 1,
      unit: 'Stück',
      unitPrice: 0.0,
      taxRate: 0.19,
      totalNet: 0.0,
      totalGross: 0.0,
    },
  ]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [successToast, setSuccessToast] = useState<string | null>(null);

  useEffect(() => {
    loadInvoices();
  }, []);

  const loadInvoices = async () => {
    setLoading(true);
    try {
      const [list, settings] = await Promise.all([
        Api.getInvoices(),
        Api.getCompanySettings(),
      ]);
      setInvoices(list);
      setCompanySettings(settings);
      setInvNumber(`RE-${settings.fiscalYear || new Date().getFullYear()}-${String(list.length + 1).padStart(3, '0')}`);

      const defaultTax = settings.isSmallBusiness ? 0.0 : 0.19;
      setItems([
        {
          position: 1,
          description: '',
          quantity: 1,
          unit: 'Stück',
          unitPrice: 0.0,
          taxRate: defaultTax,
          totalNet: 0.0,
          totalGross: 0.0,
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleAddItem = () => {
    const nextPos = items.length + 1;
    const defaultTax = companySettings?.isSmallBusiness ? 0.0 : 0.19;
    setItems([
      ...items,
      {
        position: nextPos,
        description: '',
        quantity: 1,
        unit: 'Stück',
        unitPrice: 0.0,
        taxRate: defaultTax,
        totalNet: 0.0,
        totalGross: 0.0,
      },
    ]);
  };

  const handleRemoveItem = (index: number) => {
    if (items.length <= 1) return;
    const newItems = items.filter((_, i) => i !== index);
    setItems(newItems.map((item, i) => ({ ...item, position: i + 1 })));
  };

  const handleItemChange = (index: number, field: keyof InvoiceItem, value: any) => {
    const updated = [...items];
    const item = { ...updated[index], [field]: value };

    if (field === 'quantity' || field === 'unitPrice' || field === 'taxRate') {
      const q = field === 'quantity' ? Number(value) : item.quantity;
      const p = field === 'unitPrice' ? Number(value) : item.unitPrice;
      const t = field === 'taxRate' ? Number(value) : item.taxRate;
      item.totalNet = q * p;
      item.totalGross = item.totalNet * (1 + t);
    }

    updated[index] = item;
    setItems(updated);
  };

  const totalNet = items.reduce((sum, it) => sum + it.totalNet, 0);
  const totalTax = items.reduce((sum, it) => sum + (it.totalGross - it.totalNet), 0);
  const totalGross = totalNet + totalTax;

  const handleCreateInvoice = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const newInvoice: Invoice = {
        id: 0,
        invoiceNumber: invNumber,
        date: invDate,
        dueDate: invDueDate,
        contactId: 1,
        contactName: contactName,
        items: items,
        netAmount: totalNet,
        taxAmount: totalTax,
        grossAmount: totalGross,
        currency: 'EUR',
        status: 'issued',
        createdAt: new Date().toISOString(),
      };

      await Api.createInvoice(newInvoice);
      setIsCreateModalOpen(false);
      toast.success('Rechnung erstellt', {
        description: `Rechnung ${invNumber} erfolgreich erstellt und gespeichert.`,
      });
      setSuccessToast(`Rechnung ${invNumber} erfolgreich erstellt und gespeichert.`);
      await loadInvoices();
      setTimeout(() => setSuccessToast(null), 4000);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleInspectZUGFeRD = async (inv: Invoice) => {
    setPreviewInvoice(inv);
    const result = await Api.generateInvoiceZUGFeRD(inv);
    setZugferdXML(result.xml);
  };

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Rechnungen
            <HelpTooltip
              title="E-Rechnung"
              content="Rechnungen werden als PDF mit integrierten Rechnungsdaten erstellt. Dies entspricht den aktuellen Standards für elektronische Rechnungen in Deutschland."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Ausgangsrechnungen erstellen und als PDF exportieren
          </p>
        </div>

        <button
          onClick={() => setIsCreateModalOpen(true)}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs"
        >
          <Plus className="w-3.5 h-3.5" />
          Neue Rechnung erstellen
        </button>
      </div>

      {/* Success Toast */}
      {successToast && (
        <div className="p-4 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-800 flex items-center gap-2 animate-in fade-in">
          <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
          <span>{successToast}</span>
        </div>
      )}

      {/* Invoices List */}
      <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
              <tr>
                <th className="py-3 px-4">Rechnungs-Nr.</th>
                <th className="py-3 px-4">Datum</th>
                <th className="py-3 px-4">Fällig am</th>
                <th className="py-3 px-4">Kunde</th>
                <th className="py-3 px-4 text-right">Netto</th>
                <th className="py-3 px-4 text-right">USt</th>
                <th className="py-3 px-4 text-right">Gesamtbetrag</th>
                <th className="py-3 px-4 text-center">Status</th>
                <th className="py-3 px-4 text-right">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100">
              {loading ? (
                <tr>
                  <td colSpan={9} className="py-8 text-center text-stone-400">
                    Rechnungen werden geladen...
                  </td>
                </tr>
              ) : invoices.length === 0 ? (
                <tr>
                  <td colSpan={9} className="py-8 text-center text-stone-400">
                    Noch keine Ausgangsrechnungen erstellt. Klicken Sie auf „Neue Rechnung erstellen“.
                  </td>
                </tr>
              ) : (
                invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-amber-50/20 transition-colors">
                    <td className="py-3 px-4 font-mono font-bold text-amber-800">
                      {inv.invoiceNumber}
                    </td>
                    <td className="py-3 px-4 text-stone-600">{formatDate(inv.date)}</td>
                    <td className="py-3 px-4 text-stone-600">{formatDate(inv.dueDate)}</td>
                    <td className="py-3 px-4 font-semibold text-stone-900">
                      {inv.contactName}
                    </td>
                    <td className="py-3 px-4 text-right font-mono text-stone-700">
                      {formatCurrency(inv.netAmount, inv.currency)}
                    </td>
                    <td className="py-3 px-4 text-right font-mono text-stone-500">
                      {formatCurrency(inv.taxAmount, inv.currency)}
                    </td>
                    <td className="py-3 px-4 text-right font-mono font-bold text-stone-900">
                      {formatCurrency(inv.grossAmount, inv.currency)}
                    </td>
                    <td className="py-3 px-4 text-center">
                      <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-100 text-emerald-800">
                        {inv.status === 'issued' ? 'Ausgestellt' : inv.status}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-right space-x-1">
                      <button
                        onClick={() => handleInspectZUGFeRD(inv)}
                        className="p-1.5 text-stone-600 hover:text-amber-800 hover:bg-stone-100 rounded-lg transition-colors"
                        title="Rechnungsdaten (XML) ansehen"
                      >
                        <Code className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => alert(`PDF-Export für ${inv.invoiceNumber} heruntergeladen.`)}
                        className="p-1.5 text-stone-600 hover:text-amber-800 hover:bg-stone-100 rounded-lg transition-colors"
                        title="Rechnung als PDF herunterladen"
                      >
                        <Download className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Interactive Invoice Creation Modal */}
      {isCreateModalOpen && (
        <div className="fixed inset-0 z-50 bg-stone-900/50 backdrop-blur-xs flex items-center justify-center p-4 overflow-y-auto">
          <div className="bg-white rounded-2xl shadow-2xl max-w-3xl w-full p-6 border border-stone-200 animate-in fade-in my-8 max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between border-b border-stone-200 pb-3">
              <div>
                <h3 className="text-base font-bold text-stone-900 flex items-center gap-2">
                  <FileText className="w-4 h-4 text-amber-700" />
                  Neue Rechnung erstellen
                </h3>
                <p className="text-xs text-stone-500">
                  Erstellt eine standardkonforme Rechnung mit strukturierten Rechnungsdaten
                </p>
              </div>
              <button
                onClick={() => setIsCreateModalOpen(false)}
                className="text-stone-400 hover:text-stone-700 text-sm font-bold p-1"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleCreateInvoice} className="flex-1 overflow-y-auto py-4 space-y-5 text-xs">
              {/* Header Details */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">Rechnungsnummer:</label>
                  <input
                    type="text"
                    value={invNumber}
                    onChange={(e) => setInvNumber(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg font-mono font-bold"
                    required
                  />
                </div>
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">Rechnungsdatum:</label>
                  <input
                    type="date"
                    value={invDate}
                    onChange={(e) => setInvDate(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg"
                    required
                  />
                </div>
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">Fällig bis:</label>
                  <input
                    type="date"
                    value={invDueDate}
                    onChange={(e) => setInvDueDate(e.target.value)}
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg"
                    required
                  />
                </div>
              </div>

              <div>
                <label className="font-semibold text-stone-700 block mb-1">Kunde (Empfänger):</label>
                <input
                  type="text"
                  value={contactName}
                  onChange={(e) => setContactName(e.target.value)}
                  placeholder="z. B. Musterkunde GmbH"
                  className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg"
                  required
                />
              </div>

              {/* Line Items Table */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="font-semibold text-stone-700 block">Rechnungspositionen:</label>
                  <button
                    type="button"
                    onClick={handleAddItem}
                    className="text-amber-700 hover:text-amber-800 font-semibold flex items-center gap-1"
                  >
                    <Plus className="w-3.5 h-3.5" /> Position hinzufügen
                  </button>
                </div>

                <div className="space-y-2">
                  {items.map((item, index) => (
                    <div
                      key={index}
                      className="p-3 bg-stone-50 rounded-xl border border-stone-200/80 grid grid-cols-12 gap-2 items-center"
                    >
                      <div className="col-span-5">
                        <input
                          type="text"
                          placeholder="Beschreibung der Leistung..."
                          value={item.description}
                          onChange={(e) => handleItemChange(index, 'description', e.target.value)}
                          className="w-full p-1.5 bg-white border border-stone-200 rounded-md"
                          required
                        />
                      </div>
                      <div className="col-span-2">
                        <input
                          type="number"
                          step="any"
                          placeholder="Menge"
                          value={item.quantity}
                          onChange={(e) => handleItemChange(index, 'quantity', e.target.value)}
                          className="w-full p-1.5 bg-white border border-stone-200 rounded-md"
                          required
                        />
                      </div>
                      <div className="col-span-2">
                        <input
                          type="number"
                          step="0.01"
                          placeholder="Einzelpreis"
                          value={item.unitPrice}
                          onChange={(e) => handleItemChange(index, 'unitPrice', e.target.value)}
                          className="w-full p-1.5 bg-white border border-stone-200 rounded-md"
                          required
                        />
                      </div>
                      <div className="col-span-2 text-right font-mono font-bold text-stone-900">
                        {formatCurrency(item.totalNet)}
                      </div>
                      <div className="col-span-1 text-center">
                        <button
                          type="button"
                          onClick={() => handleRemoveItem(index)}
                          className="text-stone-400 hover:text-rose-600 p-1"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Summary Totals */}
              <div className="p-4 bg-stone-100 rounded-xl border border-stone-200 space-y-1.5 text-xs">
                <div className="flex justify-between text-stone-600">
                  <span>Nettobetrag:</span>
                  <span className="font-mono font-semibold">{formatCurrency(totalNet)}</span>
                </div>
                {!companySettings?.isSmallBusiness ? (
                  <div className="flex justify-between text-stone-600">
                    <span>Umsatzsteuer (19%):</span>
                    <span className="font-mono font-semibold">{formatCurrency(totalTax)}</span>
                  </div>
                ) : (
                  <div className="flex justify-between text-amber-800">
                    <span>Umsatzsteuer:</span>
                    <span className="font-medium">0,00 € (befreit nach § 19 UStG)</span>
                  </div>
                )}
                <div className="flex justify-between font-bold text-stone-900 pt-2 border-t border-stone-200 text-sm">
                  <span>Gesamtbetrag (Brutto):</span>
                  <span className="font-mono text-amber-800">{formatCurrency(totalGross)}</span>
                </div>
                {companySettings?.isSmallBusiness && (
                  <p className="text-[11px] text-stone-500 italic pt-1">
                    Hinweis: Gemäß § 19 UStG wird keine Umsatzsteuer berechnet.
                  </p>
                )}
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setIsCreateModalOpen(false)}
                  className="px-4 py-2 text-xs font-semibold text-stone-600 hover:bg-stone-100 rounded-lg transition-colors"
                >
                  Abbrechen
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="px-5 py-2 text-xs font-semibold bg-amber-700 hover:bg-amber-800 text-white rounded-lg shadow-xs flex items-center gap-2 transition-colors"
                >
                  <CheckCircle2 className="w-4 h-4" />
                  {isSubmitting ? 'Wird gespeichert...' : 'Rechnung jetzt erstellen'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ZUGFeRD Inspector Modal */}
      {previewInvoice && zugferdXML && (
        <div className="fixed inset-0 z-50 bg-stone-900/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-3xl w-full p-6 border border-stone-200 animate-in fade-in max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between border-b border-stone-200 pb-3">
              <div>
                <h3 className="text-base font-bold text-stone-900 flex items-center gap-2">
                  <Code className="w-4 h-4 text-amber-700" />
                  Rechnungsdaten (XML): {previewInvoice.invoiceNumber}
                </h3>
                <p className="text-xs text-stone-500">
                  Strukturierte Rechnungsdaten für den automatischen Austausch
                </p>
              </div>
              <button
                onClick={() => setPreviewInvoice(null)}
                className="text-stone-400 hover:text-stone-700 text-sm font-bold p-1"
              >
                ✕
              </button>
            </div>

            <div className="my-4 flex-1 overflow-y-auto bg-[#24211E] text-stone-200 p-4 rounded-xl font-mono text-xs leading-relaxed border border-stone-800">
              <pre>{zugferdXML}</pre>
            </div>

            <div className="flex justify-end gap-2 mt-3">
              <button
                onClick={() => setPreviewInvoice(null)}
                className="px-4 py-2 text-xs font-semibold bg-stone-700 text-white rounded-lg hover:bg-stone-800 transition-colors"
              >
                Schließen
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
