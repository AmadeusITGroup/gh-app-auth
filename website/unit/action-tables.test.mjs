import assert from 'node:assert/strict';
import test from 'node:test';
import { checkActionTables } from '../scripts/lib/action-tables.mjs';

const inputs = (rows) => `### Inputs\n\n| Input | Required | Default | Description |\n| --- | --- | --- | --- |\n${rows.join('\n')}\n`;
const outputs = (rows) => `### Outputs\n\n| Output | Description |\n| --- | --- |\n${rows.join('\n')}\n`;
const metadata = {
  inputs: { 'app-id': { default: '' }, 'cleanup-on-exit': { default: 'true' }, 'app-name': { default: 'GitHub App' }, API_URL: {}, key_1: {} },
  outputs: { 'config-path': {}, DONE_FLAG: {} },
};
const goodInputs = inputs([
  '| `app-id` | Yes | none | ID |',
  '| `cleanup-on-exit` | No | `true` | Cleanup |',
  '| `app-name` | No | `GitHub App` | Name |',
  '| `API_URL` | No | none | URL |',
  '| `key_1` | No | none | Key |',
]);
const goodOutputs = outputs(['| `config-path` | Path |', '| `DONE_FLAG` | Flag |']);
const run = (text = goodInputs, out = goodOutputs, meta = metadata) => checkActionTables('setup', meta, text, out);

test('valid input and output tables, including uppercase and underscore names, pass', () => {
  assert.deepEqual(run(), []);
});

test('missing and extra identifiers are reported for inputs and outputs', () => {
  assert.ok(run(goodInputs.replace('| `key_1` | No | none | Key |\n', '')).some((p) => /input "key_1" is not documented/.test(p)));
  assert.ok(run(goodInputs + '| `extra` | No | none | x |\n').some((p) => /documented input "extra"/.test(p)));
  assert.ok(run(goodInputs, outputs(['| `config-path` | Path |'])).some((p) => /output "DONE_FLAG" is not documented/.test(p)));
});

test('a default must match exactly in the Default column', () => {
  assert.ok(run(goodInputs.replace('`true`', '`true-ish`')).some((p) => /cleanup-on-exit/.test(p)));
  const booleanFalse = { ...metadata, inputs: { ...metadata.inputs, 'cleanup-on-exit': { default: false } } };
  assert.deepEqual(run(goodInputs.replace('`true`', '`false`'), goodOutputs, booleanFalse), []);
  assert.ok(run(goodInputs.replace('`true`', '`false-ish`'), goodOutputs, booleanFalse).some((p) => /cleanup-on-exit/.test(p)));
  assert.ok(run(goodInputs.replace('| `true` | Cleanup |', '| none | default is true |')).some((p) => /cleanup-on-exit/.test(p)));
  assert.ok(run(goodInputs.replace('`GitHub App`', '`GitHub App ` ')).some((p) => /app-name/.test(p)));
});

test('backticked boolean and strings with spaces are accepted, and empty defaults are out of scope', () => {
  assert.deepEqual(run(), []);
  assert.deepEqual(run(goodInputs.replace('| `app-id` | Yes | none | ID |', '| `app-id` | Yes | anything | ID |')), []);
});

test('a missing Default header or cell is not accepted', () => {
  const noHeader = goodInputs.replace('| Input | Required | Default | Description |', '| Input | Required | Value | Description |');
  assert.ok(run(noHeader).some((p) => /Default column/.test(p)));
  const noCell = goodInputs.replace('| `cleanup-on-exit` | No | `true` | Cleanup |', '| `cleanup-on-exit` | No |');
  assert.ok(run(noCell).some((p) => /cleanup-on-exit/.test(p)));
});

test('the Default column is required only when an input has a nonempty default', () => {
  const nameOnly = '### Inputs\n\n| Input | Description |\n| --- | --- |\n| `a` | A |\n';
  assert.deepEqual(checkActionTables('x', { inputs: { a: { default: '' }, b: {} }, outputs: {} }, nameOnly.replace('| `a` | A |', '| `a` | A |\n| `b` | B |'), ''), []);
  assert.deepEqual(checkActionTables('x', { outputs: { o: {} } }, '', goodOutputsFor('o')), []);
  assert.ok(checkActionTables('x', { inputs: { a: { default: 'v' } }, outputs: {} }, nameOnly, '').some((p) => /Default column/.test(p)));
});

function goodOutputsFor(name) {
  return `### Outputs\n\n| Output | Description |\n| --- | --- |\n| \`${name}\` | O |\n`;
}
