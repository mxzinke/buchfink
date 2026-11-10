import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./takeover.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { buildTakeoverRequest, takeoverTotals } = await import(
  `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`
);

const parse = (text) => {
  const value = Number(text.replace(/\./g, '').replace(',', '.'));
  return text.trim() && Number.isFinite(value) ? Math.round(value * 100) : null;
};

const emptyItem = { contactId: 0, documentNumber: '', documentDate: '', dueDate: '', amount: '', tax: '' };
const emptyAsset = {
  name: '',
  account: '',
  acquisitionDate: '',
  acquisitionCost: '',
  accumulatedDepreciation: '',
  usefulLifeMonths: '',
  method: '',
};

const draft = {
  receiptId: 7,
  legacySystem: ' Vorgängerprogramm ',
  accounts: [
    { account: '0520', side: 'S', amount: '20.000,00' },
    { account: '1200', side: 'S', amount: '1.190,00' },
    { account: '2900', side: 'H', amount: '21.190,00' },
    { account: '', side: 'S', amount: '' },
  ],
  receivables: [
    { contactId: 3, documentNumber: 'RE-98', documentDate: '2025-12-10', dueDate: '', amount: '1.190,00', tax: '19' },
    emptyItem,
  ],
  payables: [],
  assets: [
    {
      name: 'Pkw',
      account: '0520',
      acquisitionDate: '2024-01-15',
      acquisitionCost: '30.000,00',
      accumulatedDepreciation: '10.000,00',
      usefulLifeMonths: '72',
      method: '',
    },
    emptyAsset,
  ],
};

test('Zeilen werden zur Schlussbilanz, leere entfallen', () => {
  const { request, error } = buildTakeoverRequest(draft, parse);
  assert.equal(error, undefined);
  assert.deepEqual(request, {
    receiptId: 7,
    legacySystem: 'Vorgängerprogramm',
    accounts: [
      { account: '0520', side: 'S', amount: 2_000_000 },
      { account: '1200', side: 'S', amount: 119_000 },
      { account: '2900', side: 'H', amount: 2_119_000 },
    ],
    receivables: [
      {
        contactId: 3,
        amount: 119_000,
        documentNumber: 'RE-98',
        documentDate: '2025-12-10',
        dueDate: undefined,
        taxRate: 1900,
        taxTreatment: 'domestic',
      },
    ],
    payables: [],
    assets: [
      {
        name: 'Pkw',
        account: '0520',
        acquisitionDate: '2024-01-15',
        acquisitionCost: 3_000_000,
        accumulatedDepreciation: 1_000_000,
        usefulLifeMonths: 72,
        method: undefined,
      },
    ],
  });
});

test('ein offener Posten ohne Steuerfall bleibt ohne Steuerdaten', () => {
  const { request } = buildTakeoverRequest(
    { ...draft, receivables: [{ ...draft.receivables[0], tax: '' }] },
    parse,
  );
  assert.equal(request.receivables[0].taxRate, undefined);
  assert.equal(request.receivables[0].taxTreatment, undefined);
});

test('unvollständige Zeilen nennt der Fehler mit ihrer Stelle', () => {
  assert.match(buildTakeoverRequest({ ...draft, receiptId: 0 }, parse).error, /Beleg mit der Schlussbilanz/);
  assert.equal(
    buildTakeoverRequest({ ...draft, accounts: [{ account: '1800', side: 'S', amount: '' }] }, parse).error,
    'Bestände, Zeile 1: Der Betrag fehlt.',
  );
  assert.equal(
    buildTakeoverRequest({ ...draft, payables: [{ ...emptyItem, contactId: 4, amount: '10,00' }] }, parse).error,
    'Verbindlichkeiten, Zeile 1: Die Rechnungsnummer fehlt.',
  );
  assert.equal(
    buildTakeoverRequest(
      { ...draft, assets: [{ ...draft.assets[0], accumulatedDepreciation: '31.000,00' }] },
      parse,
    ).error,
    'Anlagen, Zeile 1: Die bisherige Abschreibung liegt zwischen null und den Anschaffungskosten.',
  );
});

test('Summen der Bestände je Seite', () => {
  assert.deepEqual(takeoverTotals(draft.accounts, parse), { debit: 2_119_000, credit: 2_119_000 });
});
