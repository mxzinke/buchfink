import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./norms.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2020 } }).outputText;
const { splitNorms } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

test('takes norms in parentheses out of the visible text', () => {
  assert.deepEqual(splitNorms('Verpflichtungen, deren Höhe noch offen ist (§ 249 HGB).'), {
    text: 'Verpflichtungen, deren Höhe noch offen ist.',
    norms: ['§ 249 HGB'],
  });
  const linear = splitNorms(
    'Lineare Abschreibung über die Nutzungsdauer (§ 7 Abs. 1 EStG), zeitanteilig ab dem Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG).',
  );
  assert.equal(linear.text, 'Lineare Abschreibung über die Nutzungsdauer, zeitanteilig ab dem Anschaffungsmonat.');
  assert.deepEqual(linear.norms, ['§ 7 Abs. 1 EStG', '§ 7 Abs. 1 Satz 4 EStG']);
});

test('takes a trailing "nach § …" out', () => {
  assert.deepEqual(splitNorms('Schädlicher Beteiligungserwerb nach § 8c KStG'), {
    text: 'Schädlicher Beteiligungserwerb',
    norms: ['§ 8c KStG'],
  });
});

test('keeps parentheses that say more than a norm', () => {
  const text = 'Sofortabzug (bis 800 €) im Jahr der Anschaffung';
  assert.deepEqual(splitNorms(text), { text, norms: [] });
});
