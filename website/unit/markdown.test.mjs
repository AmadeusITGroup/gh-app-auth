import assert from 'node:assert/strict';
import test from 'node:test';
import { extractLinks, frontmatterOf, headingAnchors } from '../scripts/lib/markdown.mjs';

const sample = `---
title: T
description: D
---

# Top \`code\` heading

See [a](./a.md#x) and \`[no](./skip.md)\`.

\`\`\`md
[ignored](./nope.md)
# not a heading
\`\`\`

## Top \`code\` heading

[ref]: ./ref.md
`;

test('extractLinks ignores code and reports reference definitions', () => {
  assert.deepEqual(
    extractLinks(sample).map((link) => link.href),
    ['./a.md#x', './ref.md'],
  );
});

test('headingAnchors uses github slugs with duplicate suffixes and skips fences', () => {
  assert.deepEqual([...headingAnchors(sample)], ['top-code-heading', 'top-code-heading-1']);
});

test('frontmatterOf reads simple fields', () => {
  assert.deepEqual(frontmatterOf(sample), { title: 'T', description: 'D' });
  assert.deepEqual(frontmatterOf('# no frontmatter'), {});
});

test('fences close only with the same character, a run at least as long and a blank tail', () => {
  const text = [
    '````text',
    '[inner](missing.md)',
    '```',
    '[still inner](missing.md)',
    '```text',
    '[also inner](missing.md)',
    '````',
    '[outside](real.md)',
    '~~~',
    '[tilde](missing.md)',
    '````',
    '[tilde still](missing.md)',
    '~~~~  ',
    '[after](after.md)',
  ].join('\n');
  assert.deepEqual(
    extractLinks(text).map((link) => [link.href, link.line]),
    [['real.md', 8], ['after.md', 14]],
  );
});

test('a backtick fence with a backtick in its info is not a fence and four-space indent is code', () => {
  const links = extractLinks('```yaml`x\n[kept](kept.md)\n```\n');
  assert.deepEqual(links.map((link) => link.href), ['kept.md']);
  const indented = extractLinks('    ```\n[kept](kept.md)\n');
  assert.deepEqual(indented.map((link) => link.href), ['kept.md']);
});

test('CRLF fences are recognised and the closing fence is found', () => {
  const crlf = '````text\r\n[inside](missing.md)\r\n# Hidden\r\n````\r\n[outside](real.md)\r\n';
  assert.deepEqual(extractLinks(crlf).map((link) => [link.href, link.line]), [['real.md', 5]]);
  assert.ok(!headingAnchors(crlf).has('hidden'));
});

test('a fence closing line with a non-breaking space tail does not close the fence', () => {
  const nonblank = '````text\n````\u00a0\n[inside](missing.md)\n# Hidden\n````\n[outside](real.md)\n';
  assert.deepEqual(extractLinks(nonblank).map((link) => [link.href, link.line]), [['real.md', 6]]);
  assert.ok(!headingAnchors(nonblank).has('hidden'));
});
