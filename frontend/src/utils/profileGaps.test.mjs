import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./profileGaps.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { profileGaps, profileTask } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

const complete = {
  seat: 'Hamburg',
  managingDirectors: 'Jana Petersen',
  registerCourt: 'Amtsgericht Hamburg',
  registerNumber: 'HRB 1',
  street: 'Speicherstadt',
  postalCode: '20457',
  city: 'Hamburg',
  taxNumber: '27/123/45678',
  federalState: 'HH',
  vatPeriod: 'monthly',
  shareCapital: 2500000,
  shareholders: [{ name: 'Jana Petersen', shareCapital: 2500000 }],
  foundedOn: '2019-03-01',
};
const facts = { bankAccounts: 1, hasFoundation: false, firstYear: 2024 };

test('a complete profile has no task', () => {
  assert.deepEqual(profileGaps(complete, facts), []);
  assert.equal(profileTask([]), null);
});

test('lists what only the human can enter, in block order', () => {
  const gaps = profileGaps(
    { ...complete, taxNumber: '', street: '', vatPeriod: 'unknown', foundedOn: '2025-02-01' },
    { ...facts, bankAccounts: 0 },
  );
  assert.deepEqual(gaps, ['Straße', 'Steuernummer', 'Voranmeldungszeitraum', 'Bankkonto', 'Gründung']);
  const task = profileTask(gaps);
  assert.equal(task.title, 'Stammdaten vervollständigen: Anschrift, Steuernummer, Voranmeldungszeitraum, Bankkonto, Gründung');
  assert.deepEqual(task.target, { area: 'company', tab: 'profile' });
});

test('a company in founding has no register entry yet', () => {
  assert.deepEqual(profileGaps({ ...complete, inGruendung: true, registerNumber: '' }, facts), []);
});
