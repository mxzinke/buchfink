import { call } from './rpc.mjs';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { execFileSync } from 'node:child_process';

// Run once against the isolated bughunt server. Every write uses synthetic data.
const out = resolve('.cache/bughunt-2026-09-12/evidence/year-end');
mkdirSync(out, { recursive: true });
const steps = existsSync(`${out}/setup.json`) ? JSON.parse(readFileSync(`${out}/setup.json`)) : [];
async function run(method, ...args) {
  const previous = steps.find(step => step.method === method &&
    JSON.stringify(step.args) === JSON.stringify(args) && !step.error);
  if (previous) return previous.result;
  try {
    const result = await call(method, ...args);
    steps.push({ method, args, result });
    writeFileSync(`${out}/setup.json`, JSON.stringify(steps, null, 2));
    console.log(`${method}: OK`);
    return result;
  } catch (error) {
    steps.push({ method, args, error: error.message });
    writeFileSync(`${out}/setup.json`, JSON.stringify(steps, null, 2));
    throw error;
  }
}

const settings = steps.find(step => step.method === 'CreateTenant')?.args[2]
  ?? await call('GetCompanySettings');
const tenant = await run('CreateTenant', 'Nordlicht Beratung UG (haftungsbeschränkt)',
  '/tmp/buchfink-bughunt-2026-09-12/year-2025', {
    ...settings, companyName: 'Nordlicht Beratung UG (haftungsbeschränkt)',
    fiscalYear: 2025, contactName: 'Nora Beispiel',
    invoiceCheckSince: '2025-01-01', inGruendung: true,
  });
await call('SwitchTenant', tenant.id);
// Observed onboarding defect: a newly selected historical year is not active.
// Select and create it explicitly so the rest of the closing can be exercised.
await call('SetFiscalYear', 2025);
await run('CreateFiscalYear', 2025);
await run('SaveFoundation', {
  notarizedOn: '2025-01-15', shareCapital: 500000, foundationCostCap: 30000,
  shareholders: [{ name: 'Nora Beispiel', shareCapital: 500000, paidIn: 500000, kind: 'cash' }],
});
await run('BookFoundationPostings');
await run('RegisterCompany', '2025-01-20', 'Amtsgericht Berlin-Charlottenburg', 'HRB 234567 B');
await run('FileOpeningBalance');
const customer = await run('SaveContact', {
  type: 'customer', name: 'Beispielprojekt GmbH', street: 'Beispielweg 4',
  postalCode: '10405', city: 'Berlin', countryCode: 'DE',
  email: 'projekt@example.invalid', paymentTermsDays: 14,
});
const invoice = await run('IssueInvoice', {
  fiscalYear: 2025, date: '2025-06-30', serviceDateFrom: '2025-06-01',
  serviceDateTo: '2025-06-30', dueDate: '2025-07-14', contactId: customer.id,
  currency: 'EUR', taxTreatment: 'domestic',
  items: [{ position: 1, description: 'Konzept und Umsetzung einer Website',
    quantityMilli: 1000, unit: 'C62', unitPrice: 1000000, taxRate: 1900 }],
});
await run('SettlePayment', {
  paymentAccount: '1800', paymentDate: '2025-07-05',
  description: 'Beispielprojekt bezahlt die Website',
  allocations: [{ openItemEntryId: invoice.journalEntryId, settledAmount: 1190000 }],
});
const vendor = await run('SaveContact', {
  type: 'vendor', name: 'Beispielbedarf GmbH', street: 'Musterallee 2',
  postalCode: '10405', city: 'Berlin', countryCode: 'DE',
  taxId: '112/123/45678', paymentTermsDays: 14,
});

async function purchase(name, date, serviceFrom, serviceTo, net, account, description) {
  const tax = Math.round(net * 0.19);
  const amount = cents => (cents / 100).toFixed(2).replace('.', ',');
  writeFileSync(`${out}/${name}.typ`, `#set page(paper: "a4", margin: 24mm)
#set text(lang: "de", size: 11pt)
= Rechnung TEST-${name}
Beispielbedarf GmbH, Musterallee 2, 10405 Berlin \\
Steuernummer 112/123/45678

An Nordlicht Beratung UG (haftungsbeschränkt), Musterweg 12, 10405 Berlin

Rechnungsdatum: ${date} \\
Leistungszeitraum: ${serviceFrom} bis ${serviceTo}

${description}

Netto: ${amount(net)} Euro \\
Umsatzsteuer 19 %: ${amount(tax)} Euro \\
Gesamt: ${amount(net + tax)} Euro

Beispieldaten für einen Softwaretest.
`);
  execFileSync('typst', ['compile', `${out}/${name}.typ`, `${out}/${name}.pdf`]);
  const receipt = await run('FileIncomingReceipt', date, 'email', [{ path: `${out}/${name}.pdf`, role: 'original' }]);
  await run('SaveReceiptHeader', receipt.id, {
    kind: 'invoice', documentDate: date, issuerName: vendor.name,
    grossAmount: net + tax, taxAmount: tax, currency: 'EUR', subject: description,
  });
  await run('SaveServiceProof', receipt.id, `Softwaretest: ${description} mit Bestellung und Lieferung abgeglichen`, date);
  return run('PostIncomingReceipt', {
    contactId: vendor.id, receiptId: receipt.id, bookingDate: date, documentDate: date,
    serviceDateFrom: serviceFrom, serviceDateTo: serviceTo, description,
    taxTreatment: 'domestic', positions: [{ account, net, taxRate: 1900 }],
    settlement: 'paid', paymentAccount: '1800',
  });
}

const furniture = await purchase('schreibtisch', '2025-07-01', '2025-07-01', '2025-07-01',
  120000, '0650', 'Ein höhenverstellbarer Schreibtisch');
await run('SaveFixedAsset', {
  name: 'Höhenverstellbarer Schreibtisch', class: 'tangible', account: '0650',
  depreciationAccount: '6220', acquisitionDate: '2025-07-01', acquisitionCost: 120000,
  method: 'linear', usefulLifeMonths: 156, acquisitionEntryId: furniture.id,
  inputTaxAmount: 22800,
});
await run('GetDepreciationRun');
await run('BookDepreciationRun', { fiscalYear: 2025, bookingDate: '2025-12-31', assetIds: [] });
const hosting = await purchase('hosting', '2025-10-01', '2025-10-01', '2026-09-30',
  120000, '6805', 'Hosting für zwölf Monate, im Voraus bezahlt');
const accrual = {
  fiscalYear: 2025, kind: 'active', sourceEntryId: hosting.id,
  text: 'Hosting Januar bis September 2026', totalAmount: 120000,
  startDate: '2025-10-01', endDate: '2026-09-30', account: '6805',
};
await run('PreviewAccrual', accrual);
await run('BookAccrual', accrual);
await run('BookVatSettlement', 2025);
const taxes = await run('PreviewTaxProvision', 2025);
await run('BookTaxProvision', {
  fiscalYear: 2025, incomeProvision: taxes.incomeProvision,
  tradeProvision: taxes.tradeProvision, reason: 'Softwaretest: keine weiteren Hinzurechnungen oder Verlustvorträge',
});
await run('GetStatement', 2025, 'full');
await run('GetClosingSteps', 2025);
await run('RunChecks', '2025-12-31', 'year');
