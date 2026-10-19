import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

/** Übersetzt ein Modul und legt es als data-URL ab; `imports` ersetzt relative Pfade. */
function load(file, imports = {}) {
  const source = readFileSync(new URL(file, import.meta.url), 'utf8');
  let js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
  for (const [from, to] of Object.entries(imports)) js = js.replaceAll(`'${from}'`, `'${to}'`);
  return `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`;
}

const { planOverview, monthToClose, appName, onlyClaude, agentWorkSummary } = await import(
  load('./overview.ts', { './actor': load('./actor.ts'), './calendar': load('./calendar.ts') })
);

const task = (key, group, dueDate) => ({ key, group, title: key, why: '', target: { area: 'overview' }, count: 0, amount: 0, dueDate });
const obligation = (key, status, dueDate, extra = {}) => ({
  key, status, dueDate, title: key, kind: key.split('.')[0], periodKey: key.split('.')[1], ...extra,
});

test('takes duties from the task list and splits the rest into human and agent work', () => {
  const tasks = {
    today: '2026-08-25',
    overdue: [task('pflicht.ustva.2026-07', 'overdue', '2026-08-10'), task('backup_missing', 'overdue')],
    open: [
      task('bank_unmatched', 'open'),
      task('check_findings.number_gap', 'open'),
      task('check_findings.commit_overdue', 'open'),
      task('pflicht.festschreibung.2026-07', 'open', '2026-09-30'),
    ],
    upcoming: [task('pflicht.zm.2026-Q3', 'upcoming', '2026-10-26')],
  };
  const obligations = [
    obligation('ustva.2026-07', 'overdue', '2026-08-10'),
    obligation('festschreibung.2026-07', 'open', '2026-09-30'),
    obligation('zm.2026-Q3', 'open', '2026-10-26'),
    obligation('kst-erklaerung.2025', 'prepared', '2026-12-31'),
  ];
  const plan = planOverview(tasks, obligations);
  assert.deepEqual(
    plan.duties.map((d) => [d.key, d.group]),
    [
      ['ustva.2026-07', 'overdue'],
      ['festschreibung.2026-07', 'open'],
      ['kst-erklaerung.2025', 'open'],
      ['zm.2026-Q3', 'upcoming'],
    ],
  );
  assert.equal(plan.duties[0].obligation.status, 'overdue');
  assert.deepEqual(plan.agent.map((t) => t.key), ['bank_unmatched', 'check_findings.number_gap']);
  assert.deepEqual(plan.human.map((t) => t.key), ['backup_missing', 'check_findings.commit_overdue']);
  assert.equal(plan.next, undefined);
});

test('names the next obligation when nothing is due', () => {
  const tasks = { today: '2026-08-25', overdue: [], open: [], upcoming: [] };
  const plan = planOverview(tasks, [obligation('kst-erklaerung.2026', 'open', '2027-07-31')]);
  assert.equal(plan.duties.length, 0);
  assert.equal(plan.next.key, 'kst-erklaerung.2026');
});

test('closes the oldest past month with an open commitment', () => {
  const obligations = [
    obligation('festschreibung.2026-08', 'open', '2026-10-31', { periodTo: '2026-08-31' }),
    obligation('festschreibung.2026-07', 'open', '2026-09-30', { periodTo: '2026-07-31' }),
    obligation('festschreibung.2026-09', 'open', '2026-11-30', { periodTo: '2026-09-30' }),
  ];
  assert.equal(monthToClose(obligations, '2026-09-15'), '2026-07');
  assert.equal(monthToClose([], '2026-01-15'), '2025-12');
  assert.equal(monthToClose([], '2026-10-17'), '2026-09');
});

test('resolves the app behind a channel', () => {
  const apps = [{ channel: 'agent:claude-code', name: 'Claude Code' }];
  assert.equal(appName('agent:claude-code', apps), 'Claude Code');
  assert.equal(appName('agent:codex', apps), 'Codex');
  assert.equal(appName('system', apps), 'Programmlauf');
  assert.equal(onlyClaude(['agent:claude-code', 'agent:claude-desktop']), true);
  assert.equal(onlyClaude(['agent:claude-code', 'agent:codex']), false);
});

test('summarises agent work as counts', () => {
  const counted = (key, count) => ({ ...task(key, 'open'), count, target: { area: 'books', tab: key.split('_')[0] } });
  const items = agentWorkSummary([
    counted('bank_unmatched', 6),
    counted('receipts_unbooked', 2),
    counted('receipts_overdue', 1),
    counted('check_findings.number_gap', 1),
    counted('check_findings.interim_balance', 2),
  ]);
  assert.deepEqual(items.map((i) => i.label), ['6 Bankumsätze', '2 Belege, davon 1 über der Frist', '3 Prüfbefunde']);
  assert.deepEqual(agentWorkSummary([counted('bank_unmatched', 1)]).map((i) => i.label), ['1 Bankumsatz']);
});
