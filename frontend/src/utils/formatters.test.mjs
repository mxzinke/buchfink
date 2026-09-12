import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

// Load the actual formatter without a browser or an additional test framework.
const source = readFileSync(new URL('./formatters.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { parseCents, formatCents } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

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
