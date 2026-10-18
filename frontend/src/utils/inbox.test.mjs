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

const inbox = await import(load('./inbox.ts'));
const changes = await import(load('./changes.ts'));

const info = (count, oldestDays) => ({ path: '/x/Eingang', default: true, count, oldestDays, readFolders: [], exportFolder: '' });

test('the inbox context names count and age of the oldest file', () => {
  assert.equal(inbox.inboxContext(info(0, 0)), 'Leer');
  assert.equal(inbox.inboxContext(info(1, 0)), '1 Datei · seit heute');
  assert.equal(inbox.inboxContext(info(1, 1)), '1 Datei · seit gestern');
  assert.equal(inbox.inboxContext(info(3, 19)), '3 Dateien · die älteste seit 19 Tagen');
});

test('the overview reminds from fourteen days on', () => {
  assert.equal(inbox.inboxStale(info(3, 13)), false);
  assert.equal(inbox.inboxStale(info(3, 14)), true);
  assert.equal(inbox.inboxStale(info(0, 40)), false);
});

test('the default inbox lies in the data folder with its separator', () => {
  assert.equal(inbox.defaultInbox('~/Buchfink/nordlicht'), '~/Buchfink/nordlicht/Eingang');
  assert.equal(inbox.defaultInbox('/Users/anna/Buchfink/nordlicht/'), '/Users/anna/Buchfink/nordlicht/Eingang');
  assert.equal(inbox.defaultInbox('C:\\Buchfink\\nordlicht'), 'C:\\Buchfink\\nordlicht\\Eingang');
  assert.equal(inbox.defaultInbox('  '), '');
});

test('a drop reports the files added and the ones skipped with their reason', () => {
  assert.deepEqual(inbox.dropFeedback({ names: ['a.pdf'], skipped: [] }), { added: 'a.pdf liegt jetzt im Eingang.' });
  assert.deepEqual(
    inbox.dropFeedback({
      names: ['a.pdf', 'b.xml'],
      skipped: [{ name: 'brief.docx', reason: 'kein Beleg (PDF, Bild oder XML)' }],
    }),
    {
      added: '2 Dateien liegen jetzt im Eingang.',
      skipped: 'Nicht übernommen: brief.docx – kein Beleg (PDF, Bild oder XML)',
    },
  );
  assert.deepEqual(inbox.dropFeedback({ names: [], skipped: [], error: 'es ist kein Eingang eingerichtet' }), {
    error: 'Der Eingang hat die Dateien nicht angenommen: es ist kein Eingang eingerichtet',
  });
});

test('only changes of the open tenant reload, filtered by kind', () => {
  const event = { tenantId: 'nordlicht', channels: ['agent:claude-code'], kinds: ['belege', 'eingang'] };
  assert.equal(changes.forTenant(event, 'nordlicht'), true);
  assert.equal(changes.forTenant(event, 'werkbank'), false);
  assert.equal(changes.forTenant(event, undefined), false);
  assert.equal(changes.concerns(['belege', 'eingang'], ['eingang']), true);
  assert.equal(changes.concerns(['buchungen'], ['eingang']), false);
  assert.equal(changes.concerns(['buchungen'], undefined), true);
  assert.equal(changes.concerns([], ['eingang']), true);
});

test('a run of changes reloads after a pause, but not later than the maximum', () => {
  assert.equal(changes.reloadDelay(0, 0, 400, 1500), 400);
  assert.equal(changes.reloadDelay(0, 1300, 400, 1500), 200);
  assert.equal(changes.reloadDelay(0, 2000, 400, 1500), 0);
});
