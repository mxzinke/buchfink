import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

function load(file, imports = {}) {
  const source = readFileSync(new URL(file, import.meta.url), 'utf8');
  let js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
  for (const [from, to] of Object.entries(imports)) js = js.replaceAll(`'${from}'`, `'${to}'`);
  return `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`;
}
const { bankMatchPlan } = await import(load('./bankMatchPlan.ts', { './formatters': load('./formatters.ts') }));
const advance = { source: 'advance', advanceInvoiceId: 12, entryId: 0, openAmount: 11900 };
const plan = (amount, items) => bankMatchPlan({ amount, currency: 'EUR' }, { kind: 'open_item', items });

test('Voll- und Teilzahlung verwenden denselben Abschlag', () => {
  assert.deepEqual(plan(11900, [advance]), { kind: 'advance', advanceId: 12 });
  assert.deepEqual(plan(5950, [advance]), { kind: 'advance', advanceId: 12 });
  assert.equal(plan(12000, [advance]).kind, 'blocked');
});

test('Rückzahlung führt zur Bestätigung mit Begründung', () => {
  assert.deepEqual(plan(-11900, [{ ...advance, source: 'advance_refund' }]), { kind: 'refund', advanceId: 12 });
});

test('gemischte Vorschläge werden nicht versehentlich ganz als Abschlag gebucht', () => {
  assert.equal(plan(11900, [advance, { source: 'journal', entryId: 5, openAmount: 100 }]).kind, 'blocked');
});

test('gewöhnliche Rechnung behält den Zahlungsausgleich', () => {
  assert.deepEqual(plan(11900, [{ source: 'journal', entryId: 5, openAmount: 11900 }]), {
    kind: 'settle', allocations: [{ openItemEntryId: 5, settledAmount: 11900 }],
  });
});
