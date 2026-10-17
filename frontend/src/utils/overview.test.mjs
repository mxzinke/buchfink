import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./overview.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { planOverview, monthToClose, appName, onlyClaude } = await import(
  `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`
);

const task = (key, group, dueDate) => ({ key, group, title: key, why: '', target: { area: 'overview' }, count: 0, amount: 0, dueDate });
const obligation = (key, status, dueDate, extra = {}) => ({
  key, status, dueDate, title: key, kind: key.split('.')[0], periodKey: key.split('.')[1], ...extra,
});

test('takes duties from the task list and keeps the rest as work', () => {
  const tasks = {
    today: '2026-08-25',
    overdue: [task('pflicht.ustva.2026-07', 'overdue', '2026-08-10')],
    open: [task('bank_unmatched', 'open'), task('pflicht.festschreibung.2026-07', 'open', '2026-09-30')],
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
  assert.deepEqual(plan.work.map((t) => t.key), ['bank_unmatched']);
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
  assert.equal(appName('agent:codex', apps), 'codex');
  assert.equal(appName('system', apps), 'Programmlauf');
  assert.equal(onlyClaude(['agent:claude-code', 'agent:claude-desktop']), true);
  assert.equal(onlyClaude(['agent:claude-code', 'agent:codex']), false);
});
