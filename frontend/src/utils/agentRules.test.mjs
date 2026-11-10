import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

/** Übersetzt ein Modul und legt es als data-URL ab. */
function load(file) {
  const source = readFileSync(new URL(file, import.meta.url), 'utf8');
  const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
  return `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`;
}

const rules = await import(load('./agentRules.ts'));
const formatters = await import(load('./formatters.ts'));
const defaults = { maxGrossLabel: '1.000,00 €', minBookings: 3 };
const check = (maxGross, minBookings) => rules.checkRules({ maxGross, minBookings }, defaults, formatters.parseCents);

test('empty fields stand for the defaults', () => {
  assert.deepEqual(check('', ''), { directMaxGross: 0, directMinBookings: 0, errors: {} });
});

test('zero is no way to say "default" or "never"', () => {
  const zero = check('0', '0');
  assert.match(zero.errors.maxGross, /Mindestens 0,01 €.*1\.000,00 €/);
  assert.match(zero.errors.minBookings, /Mindestens 1.*3/);
  assert.equal(check('0,00', '').errors.maxGross !== undefined, true);
});

test('valid limits pass through', () => {
  assert.deepEqual(check('0,01', '1'), { directMaxGross: 1, directMinBookings: 1, errors: {} });
  assert.deepEqual(check('2.500,00', '5'), { directMaxGross: 250000, directMinBookings: 5, errors: {} });
});

test('garbage is refused', () => {
  const bad = check('viel', '2,5');
  assert.ok(bad.errors.maxGross);
  assert.ok(bad.errors.minBookings);
  assert.ok(check('-5', '').errors.maxGross);
});
