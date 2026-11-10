import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

const source = readFileSync(new URL('./actor.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { actorKind, agentName, actorLabel, channelNote } = await import(
  `data:text/javascript;base64,${Buffer.from(js).toString('base64')}`
);

test('tells person, app and program run apart by the channel', () => {
  assert.equal(actorKind('ui'), 'person');
  assert.equal(actorKind(undefined), 'person');
  assert.equal(actorKind('system'), 'system');
  assert.equal(actorKind('agent:claude-code'), 'agent');
});

test('names known apps and capitalises unknown slugs', () => {
  assert.equal(agentName('agent:claude-code'), 'Claude Code');
  assert.equal(agentName('agent:opencode'), 'OpenCode');
  assert.equal(agentName('agent:mein-agent-2'), 'Mein Agent 2');
  assert.equal(agentName('ui'), 'Agent');
});

test('says who recorded the entry', () => {
  assert.equal(actorLabel('anna@mbp', 'agent:claude-desktop'), 'Erfasst von Claude Desktop · anna@mbp');
  assert.equal(actorLabel('anna@mbp', 'ui'), 'Erfasst von anna@mbp in Buchfink');
  assert.equal(actorLabel('', 'system'), 'Erfasst von Buchfink (Programmlauf)');
});

test('names the channel only when it is not the app itself', () => {
  assert.equal(channelNote('ui'), null);
  assert.equal(channelNote(undefined), null);
  assert.equal(channelNote('system'), 'Programmlauf');
  assert.equal(channelNote('agent:claude-desktop'), 'über Claude Desktop');
});
