import React from 'react';
import type { EInvoiceView, EInvoiceViewParty } from '../../types';
import { formatDate } from '../../utils/formatters';
import { Table, Tbody, Td, Th, Thead, Tr } from '../../components/ui';

/** Ein Betrag oder eine Menge, wie die XML sie schreibt, in deutscher Schreibweise. */
function decimal(raw: string, minimumFractionDigits = 2): string {
  if (!raw) return '—';
  const value = Number(raw);
  if (Number.isNaN(value)) return raw;
  return value.toLocaleString('de-DE', { minimumFractionDigits, maximumFractionDigits: 4 });
}

function period(from?: string, to?: string): string {
  if (!from && !to) return '';
  if (from === to || !to) return formatDate(from ?? '');
  return `${formatDate(from ?? '')} – ${formatDate(to)}`;
}

const Party: React.FC<{ label: string; party: EInvoiceViewParty }> = ({ label, party }) => (
  <div>
    <p className="text-label text-ink-muted">{label}</p>
    <p className="text-body text-ink font-medium">{party.name || '—'}</p>
    {(party.address ?? []).map((line) => (
      <p key={line} className="text-body text-ink">
        {line}
      </p>
    ))}
    {party.vatId && <p className="text-caption text-ink-subtle">USt-IdNr. {party.vatId}</p>}
    {party.taxNumber && <p className="text-caption text-ink-subtle">Steuernummer {party.taxNumber}</p>}
    {party.contact && <p className="text-caption text-ink-subtle">{party.contact}</p>}
  </div>
);

const Meta: React.FC<{ label: string; value?: string }> = ({ label, value }) =>
  value ? (
    <div className="flex justify-between gap-4 text-body">
      <span className="text-ink-muted">{label}</span>
      <span className="text-ink text-right">{value}</span>
    </div>
  ) : null;

/**
 * Die Ansicht einer reinen E-Rechnung, gelesen aus der XML. Sie macht den
 * Datensatz lesbar; der Beleg ist die XML selbst.
 */
export const EInvoiceSheet: React.FC<{ invoice: EInvoiceView }> = ({ invoice }) => {
  const currency = invoice.currency || 'EUR';
  return (
    <div className="w-full self-stretch bg-surface rounded-control p-4 overflow-auto max-h-[28rem] text-left">
      <div className="flex items-baseline justify-between gap-4">
        <h3 className="text-heading text-ink">
          {invoice.typeLabel} {invoice.number}
        </h3>
        <span className="text-caption text-ink-subtle">E-Rechnung ({invoice.syntax}), Ansicht der XML</span>
      </div>

      <div className="grid grid-cols-2 gap-4 mt-4">
        <Party label="Von" party={invoice.seller} />
        <Party label="An" party={invoice.buyer} />
      </div>

      <div className="mt-4 space-y-0.5">
        <Meta label="Rechnungsdatum" value={formatDate(invoice.issueDate)} />
        <Meta label="Leistungszeitraum" value={period(invoice.periodFrom, invoice.periodTo)} />
        <Meta label="Lieferdatum" value={invoice.deliveryDate ? formatDate(invoice.deliveryDate) : ''} />
        <Meta label="Fällig am" value={invoice.dueDate ? formatDate(invoice.dueDate) : ''} />
        <Meta label="Bezug auf" value={(invoice.precedingInvoices ?? []).join(', ')} />
        <Meta label="Bestellnummer" value={invoice.orderReference} />
        <Meta label="Käuferreferenz" value={invoice.buyerReference} />
      </div>

      <Table density="kompakt" className="mt-4">
        <Thead>
          <Tr>
            <Th>Position</Th>
            <Th numeric>Menge</Th>
            <Th numeric>Preis</Th>
            <Th numeric>USt</Th>
            <Th numeric>Netto</Th>
          </Tr>
        </Thead>
        <Tbody>
          {invoice.lines.map((line) => (
            <Tr key={line.id}>
              <Td className="whitespace-normal">
                {line.name}
                {line.description && <span className="block text-caption text-ink-subtle">{line.description}</span>}
                {(line.periodFrom || line.periodTo) && (
                  <span className="block text-caption text-ink-subtle">{period(line.periodFrom, line.periodTo)}</span>
                )}
              </Td>
              <Td numeric>
                {decimal(line.quantity, 0)} {line.unit}
              </Td>
              <Td numeric>{decimal(line.netPrice)}</Td>
              <Td numeric>{line.vatRate ? `${decimal(line.vatRate, 0)} %` : line.vatCategory}</Td>
              <Td numeric>{decimal(line.netAmount)}</Td>
            </Tr>
          ))}
        </Tbody>
      </Table>

      <div className="mt-4 ml-auto max-w-xs space-y-0.5">
        <Meta label="Summe netto" value={`${decimal(invoice.totals.taxBasisTotal)} ${currency}`} />
        {invoice.vat.map((v) => (
          <Meta
            key={`${v.category}-${v.rate}`}
            label={`USt ${decimal(v.rate, 0)} % auf ${decimal(v.taxableAmount)}${v.exemptionReason ? ` (${v.exemptionReason})` : ''}`}
            value={`${decimal(v.taxAmount)} ${currency}`}
          />
        ))}
        <Meta label="Gesamt" value={`${decimal(invoice.totals.grandTotal)} ${currency}`} />
        {invoice.totals.prepaid && <Meta label="Bereits gezahlt" value={`${decimal(invoice.totals.prepaid)} ${currency}`} />}
        <Meta label="Zu zahlen" value={`${decimal(invoice.totals.duePayable)} ${currency}`} />
      </div>

      {(invoice.paymentTerms || (invoice.payment ?? []).length > 0) && (
        <div className="mt-4">
          <p className="text-label text-ink-muted">Zahlung</p>
          {invoice.paymentTerms && <p className="text-body text-ink">{invoice.paymentTerms}</p>}
          {(invoice.payment ?? []).map((p, i) => (
            <p key={i} className="text-body text-ink">
              {[p.means, p.iban, p.bic, p.accountName, p.remittanceInfo].filter(Boolean).join(' · ')}
            </p>
          ))}
        </div>
      )}

      {(invoice.notes ?? []).length > 0 && (
        <div className="mt-4">
          <p className="text-label text-ink-muted">Bemerkungen</p>
          {(invoice.notes ?? []).map((note, i) => (
            <p key={i} className="text-body text-ink whitespace-pre-line">
              {note}
            </p>
          ))}
        </div>
      )}
    </div>
  );
};
