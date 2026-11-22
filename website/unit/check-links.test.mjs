import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test, { after } from 'node:test';
import { checkMarkdownLinks } from '../scripts/lib/check-links.mjs';
import { isExternalHref, splitHref } from '../src/lib/links.mjs';
import { isDraft, listPublished } from '../src/lib/published.mjs';

const root = fs.mkdtempSync(path.join(os.tmpdir(), 'check-links-'));
after(() => fs.rmSync(root, { recursive: true, force: true }));
const write = (relative, content) => {
  const full = path.join(root, relative);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, content);
  return full;
};

const target = write('docs/guides/target.md', '# Target\n\n## Some Section\n');
write('docs/guides/with space.md', '# Spaced\n');
const source = path.join(root, 'docs/guides/source.md');
const check = (markdown) => checkMarkdownLinks(source, markdown);

test('splitHref separates path, query and hash and decodes them', () => {
  assert.deepEqual(splitHref('a%20b.md?x=1#some%20id'), {
    rawPath: 'a%20b.md',
    decodedPath: 'a b.md',
    query: '?x=1',
    hash: '#some%20id',
    anchor: 'some id',
  });
  assert.equal(splitHref('a.md').hash, '');
  assert.equal(splitHref('#only').rawPath, '');
});

test('splitHref rejects malformed percent encoding', () => {
  assert.throws(() => splitHref('a%E0%A4%A.md'), /malformed link/);
  assert.throws(() => splitHref('a.md#%E0%A4%A'), /malformed link/);
});

test('isExternalHref recognizes schemes, protocol-relative and fragment-only links', () => {
  for (const href of ['https://x.test', 'mailto:a@b.test', '//cdn.test/x', '#a', '']) {
    assert.equal(isExternalHref(href), true, href);
  }
  assert.equal(isExternalHref('./a.md'), false);
});

test('links with query and hash validate against the real file', () => {
  assert.deepEqual(check('[a](target.md?x=1#some-section)'), []);
  assert.deepEqual(check('[a](target.md#some-section)'), []);
  assert.deepEqual(check('[a](target.md?x=1)'), []);
  assert.deepEqual(check('[a](target.md)'), []);
});

test('missing files and anchors are reported even with a query', () => {
  assert.match(check('[a](missing.md?x=1#id)')[0], /missing target/);
  assert.match(check('[a](target.md?x=1#nope)')[0], /missing anchor/);
  assert.match(check('[a](target.md#nope)')[0], /missing anchor/);
});

test('encoded paths resolve and malformed parts are reported', () => {
  assert.deepEqual(check('[a](with%20space.md)'), []);
  assert.match(check('[a](bad%E0%A4%A.md)')[0], /malformed link/);
});

test('external links are not checked', () => {
  assert.deepEqual(check('[a](https://example.test/missing.md)'), []);
  assert.equal(fs.existsSync(target), true);
});

test('draft pages are excluded from the published list', () => {
  write('docs/index.mdx', '---\ntitle: Home\ndescription: x\n---\n');
  write('docs/guides/live.md', '---\ntitle: Live\ndescription: x\n---\n');
  const draft = write('docs/guides/draft.md', '---\ntitle: Draft\ndescription: x\ndraft: true\n---\n');
  assert.equal(isDraft(draft), true);
  const published = listPublished(path.join(root, 'docs'));
  assert.ok(published.includes('guides/live.md'));
  assert.ok(!published.includes('guides/draft.md'));
});

test('draft detection parses YAML: inline comments, CRLF, false and non-boolean values', () => {
  const draftFile = (name, body) => write(`docs/guides/${name}.md`, body);
  assert.equal(isDraft(draftFile('inline', '---\ntitle: T\ndraft: true # hidden for now\n---\n')), true);
  assert.equal(isDraft(draftFile('crlf', '---\r\ntitle: T\r\ndraft: true\r\n---\r\nbody\r\n')), true);
  assert.equal(isDraft(draftFile('false', '---\ntitle: T\ndraft: false\n---\n')), false);
  assert.equal(isDraft(draftFile('string', '---\ntitle: T\ndraft: "true"\n---\n')), false);
  assert.equal(isDraft(draftFile('none', '# no frontmatter\n')), false);
});

test('draft detection fails clearly on malformed frontmatter', () => {
  const broken = write('docs/guides/broken.md', '---\ntitle: [unclosed\n---\n');
  assert.throws(() => isDraft(broken), /invalid frontmatter YAML/);
});

test('fragment-only links are validated against the current file', () => {
  const page = write('docs/guides/fragments.md', '# Top\n\n## Good Section\n\n## Café\n');
  const run = (markdown) => checkMarkdownLinks(page, markdown);
  assert.deepEqual(run('[ok](#good-section)\n'), []);
  assert.match(run('[bad](#missing-section)\n')[0], /missing anchor #missing-section/);
  assert.deepEqual(run('[ok](#caf%C3%A9)\n'), []);
  assert.match(run('[bad](#caf%E0%A4%A)\n')[0], /malformed/);
  assert.deepEqual(run('[top](#)\n'), []);
});
