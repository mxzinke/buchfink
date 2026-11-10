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
const contacts = await import(load('./contacts.ts'));

const info = (count, oldestDays) => ({ path: '/x/Eingang', default: true, count, oldestDays, readFolders: [], exportFolder: '' });

test('the inbox context names count and age of the oldest file', () => {
  assert.equal(inbox.inboxContext(info(0, 0)), 'Leer');
  assert.equal(inbox.inboxContext(info(1, 0)), '1 Datei · von heute');
  assert.equal(inbox.inboxContext(info(1, 1)), '1 Datei · von gestern');
  assert.equal(inbox.inboxContext(info(3, 19)), '3 Dateien · die älteste 19 Tage alt');
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

const vendor = (id, extra = {}) => ({ id, type: 'vendor', name: `Lieferant ${id}`, company: '', blocked: false, ...extra });

test('a proposal of Claude waits for confirmation, a confirmed group does not', () => {
  assert.equal(contacts.postingGroupState(vendor(1)), 'none');
  assert.equal(contacts.postingGroupState(vendor(2, { defaultPostingGroup: 'telefon' })), 'confirmed');
  assert.equal(contacts.postingGroupState(vendor(3, { proposedPostingGroup: 'werbung' })), 'proposed');
  assert.equal(
    contacts.postingGroupState(vendor(4, { defaultPostingGroup: 'werbung', proposedPostingGroup: 'werbung' })),
    'confirmed',
  );
  assert.equal(contacts.hasPendingProposal(vendor(5, { proposedPostingGroup: 'werbung', blocked: true })), false);
  assert.equal(contacts.hasPendingProposal({ ...vendor(6, { proposedPostingGroup: 'werbung' }), type: 'customer' }), false);
});

test('the overview task leads to the one vendor or to the list', () => {
  assert.equal(contacts.postingGroupTask([vendor(1)]), null);
  const one = contacts.postingGroupTask([vendor(1), vendor(18, { proposedPostingGroup: 'fremdleistungen' })]);
  assert.equal(one.title, 'Standardgruppe für Lieferant 18 bestätigen');
  assert.deepEqual(one.target, { area: 'company', tab: 'contacts', id: '18' });
  const two = contacts.postingGroupTask([
    vendor(16, { proposedPostingGroup: 'werbung' }),
    vendor(18, { proposedPostingGroup: 'fremdleistungen' }),
  ]);
  assert.equal(two.title, '2 Standardgruppen bestätigen');
  assert.equal(two.target.id, undefined);
  assert.equal(two.count, 2);
});

test('a confirmation clears the proposal it answered', () => {
  const before = vendor(7, {
    proposedPostingGroup: 'fremdleistungen',
    proposedPostingGroupBy: 'agent:claude-code',
    openAmount: 11900,
  });
  // Die Antwort der App lässt geleerte Felder weg.
  const confirmed = { ...vendor(7), defaultPostingGroup: 'software', defaultPostingGroupConfirmedAt: '2026-10-18', openAmount: 0 };
  const after = contacts.withPostingGroupOf(before, confirmed);
  assert.equal(contacts.postingGroupState(after), 'confirmed');
  assert.equal(after.proposedPostingGroup, undefined);
  assert.equal(after.openAmount, 11900);
  const cleared = contacts.withPostingGroupOf(after, vendor(7));
  assert.equal(contacts.postingGroupState(cleared), 'none');
  assert.equal(cleared.defaultPostingGroupConfirmedAt, undefined);
});

test('only the latest load of a view sets its state', () => {
  const counter = changes.requestCounter();
  const live = counter.begin();
  const yearSwitch = counter.begin();
  // Die Antwort auf das Neuladen kommt nach der des Jahreswechsels.
  assert.equal(yearSwitch(), true);
  assert.equal(live(), false);
  const other = changes.requestCounter();
  assert.equal(other.begin()(), true);
});
