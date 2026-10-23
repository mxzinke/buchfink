import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./calendar.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { todayISO } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

test('Der Buchhaltungstag folgt Berlin an Mitternacht und bei der Zeitumstellung', () => {
  for (const [at, expected] of [
    ['2026-12-31T23:30:00Z', '2027-01-01'],
    ['2026-06-30T22:30:00Z', '2026-07-01'],
    ['2026-10-25T02:45:00+02:00', '2026-10-25'],
    ['2026-10-25T02:15:00+01:00', '2026-10-25'],
  ]) assert.equal(todayISO(new Date(at)), expected);
});
