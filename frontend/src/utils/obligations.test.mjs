import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

function load(file) {
  const source = readFileSync(new URL(file, import.meta.url), 'utf8');
  const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
  return `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`;
}

const { obligationTab, COMPANY_OBLIGATIONS } = await import(load('./obligations.ts'));

/** Die Pflichtarten aus internal/domain/obligation.go. */
const goKinds = [
  ...readFileSync(new URL('../../../internal/domain/obligation.go', import.meta.url), 'utf8').matchAll(
    /^\s+Obligation\w+\s+ObligationKind = "([^"]+)"$/gm,
  ),
].map((m) => m[1]);

// Dieselbe Zuordnung wie TestObligationTargetsAreValidRoutes in
// internal/service/task_target_test.go.
const TABS = {
  ustva: 'month',
  sondervorauszahlung: 'month',
  zm: 'month',
  'lohnsteuer-anmeldung': 'month',
  festschreibung: 'month',
  'kst-vorauszahlung': 'declarations',
  'gewst-vorauszahlung': 'declarations',
  'e-bilanz': 'declarations',
  'e-bilanz-eroeffnung': 'declarations',
  'kst-erklaerung': 'declarations',
  'gewst-erklaerung': 'declarations',
  'ust-erklaerung': 'declarations',
  abschlussbuchungen: 'year',
  aufstellung: 'year',
  feststellung: 'year',
  ergebnisverwendung: 'year',
  offenlegung: 'year',
  kapitalertragsteuer: 'year',
  transparenzregister: 'company',
  gesellschafterliste: 'company',
  'steuerliche-erfassung': 'company',
  gewerbeanmeldung: 'company',
  eroeffnungsbilanz: 'company',
};

test('every obligation kind of the core has its tab', () => {
  assert.equal(goKinds.length, 23);
  assert.deepEqual([...goKinds].sort(), Object.keys(TABS).sort());
  for (const kind of goKinds) assert.equal(obligationTab(kind), TABS[kind], kind);
});

test('the opening balance belongs to the founding, its E-Bilanz to the declarations', () => {
  assert.equal(obligationTab('eroeffnungsbilanz'), 'company');
  assert.ok(COMPANY_OBLIGATIONS.gesellschafter.includes('eroeffnungsbilanz'));
  assert.equal(obligationTab('e-bilanz-eroeffnung'), 'declarations');
});

test('a block of the company profile shows every obligation of the company tab', () => {
  const shown = [...COMPANY_OBLIGATIONS.firma, ...COMPANY_OBLIGATIONS.gesellschafter].sort();
  assert.deepEqual(shown, goKinds.filter((kind) => obligationTab(kind) === 'company').sort());
});
