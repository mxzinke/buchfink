import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

// Load the actual formatter without a browser or an additional test framework.
const source = readFileSync(new URL('./formatters.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { parseCents, formatCents, formatBalance, formatMonth } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

test('rejects malformed amounts and unsafe integers', () => {
  for (const input of ['-', '+', '.', ',', '12.34,56', '1,234.56', '1.-2', '--1', '90071992547409.92', '10,555']) {
    assert.equal(parseCents(input), null, input);
  }
});

test('preserves cents and accepts a copied formatted amount', () => {
  for (const [input, expected] of [['1.234,56', 123456], ['1234.56', 123456], [',5', 50], ['0,01', 1], ['90071992547409.91', Number.MAX_SAFE_INTEGER]]) {
    assert.equal(parseCents(input), expected, String(input));
  }
  assert.equal(parseCents(formatCents(-4250)), -4250);
});

test('marks a balance with its side', () => {
  assert.equal(formatBalance(2450000, 'asset'), '24.500,00 € S');
  assert.equal(formatBalance(2500000, 'equity'), '25.000,00 € H');
  assert.equal(formatBalance(-1452548, 'asset'), '14.525,48 € H');
  assert.equal(formatBalance(-100, 'revenue'), '1,00 € S');
  assert.equal(formatBalance(0, 'asset'), '0,00 €');
});

test('names a month', () => {
  assert.equal(formatMonth('2026-03'), 'März 2026');
  assert.equal(formatMonth('2026-12'), 'Dezember 2026');
  assert.equal(formatMonth('2026-13'), '2026-13');
  assert.equal(formatMonth(''), '—');
});
