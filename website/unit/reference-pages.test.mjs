import assert from 'node:assert/strict';
import test from 'node:test';
import { normalizeHelp, parseCommands, renderPage } from '../scripts/lib/reference-pages.mjs';

const root = `Usage:
  tool [command]

Available Commands:
  completion  Generate completion
  help        Help about any command
  setup       Configure things

Flags:
  -h, --help   help
`;

test('parseCommands lists public commands and skips help', () => {
  assert.deepEqual(parseCommands(root), [
    { name: 'completion', summary: 'Generate completion' },
    { name: 'setup', summary: 'Configure things' },
  ]);
});

test('parseCommands returns nothing for leaf commands', () => {
  assert.deepEqual(parseCommands('Usage:\n  tool setup [flags]\n'), []);
});

test('normalizeHelp trims trailing whitespace and collapses blank runs', () => {
  assert.equal(normalizeHelp('a  \r\n\n\n\nb\n'), 'a\n\nb');
});

test('normalizeHelp replaces hard tabs', () => {
  assert.equal(normalizeHelp('a\n\tindented'), 'a\n  indented');
});

test('renderPage is deterministic and has no timestamp', () => {
  const input = { name: 'setup', summary: 'Configure: things', sections: [{ title: 'setup', help: 'Usage:\n  x' }] };
  const first = renderPage(input);
  assert.equal(first, renderPage(input));
  assert.match(first, /^---\ntitle: gh app-auth setup\ndescription: Configure - things\n---/);
  assert.match(first, /```text\nUsage:\n {2}x\n```/);
  assert.match(first, /## Related/);
  assert.doesNotMatch(first, /20\d\d-\d\d-\d\d/);
  assert.match(first, /in the checked-out source/);
  assert.doesNotMatch(first, /describes v\d+\.\d+\.\d+/);
});

test('renderPage names nested subcommands and omits empty related lists', () => {
  const page = renderPage({
    name: 'unknown',
    summary: 's',
    sections: [
      { title: 'unknown', help: 'a' },
      { title: 'gh app-auth unknown child', help: 'b' },
    ],
  });
  assert.match(page, /## gh app-auth unknown child/);
  assert.doesNotMatch(page, /## Related/);
});

test('renderPage adds usage notes only for completion and setup', () => {
  const page = (name) => renderPage({ name, summary: 'S', sections: [{ title: name, help: 'Usage:\n  x' }] });
  const completion = page('completion');
  assert.match(completion, /standalone `gh-app-auth`/);
  assert.match(completion, /executable on `PATH`/);
  assert.ok(completion.indexOf('standalone') < completion.indexOf('## Help output'));
  const setup = page('setup');
  assert.match(setup, /personal-access-tokens/);
  assert.match(setup, /process arguments/);
  assert.ok(setup.indexOf('process arguments') < setup.indexOf('## Help output'));
  for (const name of ['list', 'unknown']) assert.doesNotMatch(page(name), /standalone|process arguments/);
});
