import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test, { after } from 'node:test';
import remarkSourceLinks, { rewriteHref } from '../src/plugins/source-links.mjs';
import { listPublished, routeFor } from '../src/lib/published.mjs';

const repoUrl = 'https://github.com/example/repo';
const root = fs.mkdtempSync(path.join(os.tmpdir(), 'source-links-'));
after(() => fs.rmSync(root, { recursive: true, force: true }));
const write = (relative, content = '# x\n') => {
  const full = path.join(root, relative);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, content);
};

write('docs/index.mdx');
write('docs/guides/index.md');
write('docs/guides/a.md');
write('docs/guides/ci/b.mdx');
write('docs/reference/cli/index.md');
write('docs/legacy.md');
write('docs/adr/0001.md');
write('.github/workflows/ci.yml', 'name: ci\n');
write('SECURITY.md');
fs.mkdirSync(path.join(root, 'cmd'));

const docsRoot = path.join(root, 'docs');
const published = new Set(listPublished(docsRoot));
const context = (file) => ({
  filePath: path.join(root, file),
  repoRoot: root,
  docsRoot,
  published,
  base: '/site/',
  repoUrl,
  branch: 'main',
});
const from = path.join('docs', 'guides', 'a.md');
const rewrite = (href, file = from) => rewriteHref(href, context(file));

test('published set contains sections and the home page only', () => {
  assert.deepEqual([...published], [
    'guides/a.md',
    'guides/ci/b.mdx',
    'guides/index.md',
    'index.mdx',
    'reference/cli/index.md',
  ].sort());
});

test('routeFor maps sources to routes', () => {
  assert.equal(routeFor('index.mdx'), '');
  assert.equal(routeFor('guides/index.md'), 'guides/');
  assert.equal(routeFor('guides/a.md'), 'guides/a/');
  assert.equal(routeFor('guides/ci/b.mdx'), 'guides/ci/b/');
});

test('published md and mdx sources become base-aware routes', () => {
  assert.equal(rewrite('./ci/b.mdx'), '/site/guides/ci/b/');
  assert.equal(rewrite('../index.mdx'), '/site/');
  assert.equal(rewrite('a.md'), '/site/guides/a/');
});

test('index sources, directories, fragments and queries are preserved', () => {
  assert.equal(rewrite('index.md'), '/site/guides/');
  assert.equal(rewrite('../reference/cli'), '/site/reference/cli/');
  assert.equal(rewrite('a.md#section'), '/site/guides/a/#section');
  assert.equal(rewrite('a.md?x=1#section'), '/site/guides/a/?x=1#section');
});

test('external, mailto, protocol-relative and same-page links are untouched', () => {
  for (const href of ['https://example.com/a', 'mailto:a@example.com', '//cdn.example.com/x', '#anchor', '']) {
    assert.equal(rewrite(href), href);
  }
});

test('unpublished repository files link to canonical source', () => {
  assert.equal(rewrite('../legacy.md'), `${repoUrl}/blob/main/docs/legacy.md`);
  assert.equal(rewrite('../adr/0001.md#context'), `${repoUrl}/blob/main/docs/adr/0001.md#context`);
  assert.equal(rewrite('../../.github/workflows/ci.yml'), `${repoUrl}/blob/main/.github/workflows/ci.yml`);
  assert.equal(rewrite('../../SECURITY.md'), `${repoUrl}/blob/main/SECURITY.md`);
  assert.equal(rewrite('../../cmd'), `${repoUrl}/tree/main/cmd`);
});

test('missing targets, absolute links and escapes are errors', () => {
  assert.throws(() => rewrite('missing.md'), /link target not found/);
  assert.throws(() => rewrite('./ci/b'), /link target not found/);
  assert.throws(() => rewrite('/guides/a/'), /root-absolute/);
  assert.throws(() => rewrite('../../../outside.md'), /outside the repository/);
});

test('plugin rewrites links and definitions and rejects relative images', () => {
  const plugin = remarkSourceLinks({ ...context(from), published: [...published] });
  const tree = {
    type: 'root',
    children: [
      { type: 'paragraph', children: [{ type: 'link', url: 'a.md', children: [] }] },
      { type: 'definition', url: '../legacy.md' },
    ],
  };
  plugin(tree, { path: path.join(root, from) });
  assert.equal(tree.children[0].children[0].url, '/site/guides/a/');
  assert.equal(tree.children[1].url, `${repoUrl}/blob/main/docs/legacy.md`);
  const bad = { type: 'root', children: [{ type: 'image', url: './x.png' }] };
  assert.throws(() => plugin(bad, { path: path.join(root, from) }), /relative image/);
});
