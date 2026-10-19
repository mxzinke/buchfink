import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./unterbilanz.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { unterbilanzRows } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

const base = {
  asOf: '2026-03-20',
  isFinal: true,
  shareCapital: 2_500_000,
  assets: 2_500_000,
  liabilities: 0,
  netAssets: 2_500_000,
  shortfall: 0,
  covered: 0,
  amount: 0,
  shares: [],
};

test('ohne Unterdeckung: Stammkapital, Reinvermögen und keine Haftung', () => {
  assert.deepEqual(
    unterbilanzRows(base).map((r) => [r.label, r.cents]),
    [
      ['Stammkapital', 2_500_000],
      ['Reinvermögen', 2_500_000],
      ['Haftung der Gesellschafter', 0],
    ],
  );
});

test('der Gründungsaufwand laut Satzung deckt einen Teil der Unterdeckung', () => {
  const u = { ...base, netAssets: 2_200_000, shortfall: 300_000, covered: 100_000, amount: 200_000 };
  assert.deepEqual(
    unterbilanzRows(u).map((r) => [r.label, r.cents]),
    [
      ['Stammkapital', 2_500_000],
      ['Reinvermögen', 2_200_000],
      ['Unterdeckung', 300_000],
      ['Gedeckt durch Gründungsaufwand', 100_000],
      ['Haftung der Gesellschafter', 200_000],
    ],
  );
});

test('ohne Satzungsklausel erscheint keine Zeile für den Gründungsaufwand', () => {
  const u = { ...base, netAssets: 2_200_000, shortfall: 300_000, amount: 300_000 };
  assert.ok(!unterbilanzRows(u).some((r) => r.label === 'Gedeckt durch Gründungsaufwand'));
});
