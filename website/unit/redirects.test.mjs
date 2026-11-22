import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test, { after } from 'node:test';
import { loadRedirects, validateRedirects } from '../src/lib/redirects.mjs';
import { headingAnchors } from '../scripts/lib/markdown.mjs';
import { listPublished, routeFor } from '../src/lib/published.mjs';

const created = [];
after(() => created.forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));

const writeManifest = (manifest) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'redirects-'));
  created.push(directory);
  const file = path.join(directory, 'r.json');
  fs.writeFileSync(file, JSON.stringify(manifest));
  return file;
};

test('flat redirects keep plain and .md legacy routes, the homepage and fragments under the base', () => {
  const file = writeManifest({
    '/docs/installation/': 'start-here/consolidated/#install',
    '/docs/installation.md/': 'start-here/consolidated/#install',
    '/docs/README/': '',
    '/docs/origin/': '#section',
  });
  assert.deepEqual(loadRedirects(file, '/site/'), {
    '/docs/installation/': '/site/start-here/consolidated/#install',
    '/docs/installation.md/': '/site/start-here/consolidated/#install',
    '/docs/README/': '/site/',
    '/docs/origin/': '/site/#section',
  });
});

const anchors = new Map([
  ['', new Set(['section'])],
  ['start-here/consolidated/', new Set(['install'])],
]);

test('valid routes and anchors produce no problems', () => {
  assert.deepEqual(
    validateRedirects(
      { '/docs/a/': 'start-here/consolidated/#install', '/docs/b/': '', '/docs/c/': '#section', '/old/': 'start-here/consolidated/' },
      anchors,
    ),
    [],
  );
});

test('missing targets, absent anchors and alias chains are rejected', () => {
  assert.match(validateRedirects({ '/docs/a/': 'nope/' }, anchors)[0], /not a published route/);
  assert.match(validateRedirects({ '/docs/a/': 'start-here/consolidated/#gone' }, anchors)[0], /missing anchor/);
  assert.match(validateRedirects({ '/docs/a/': '#gone' }, anchors)[0], /missing anchor/);
  const chain = validateRedirects({ '/docs/a/': 'start-here/consolidated/', '/start-here/consolidated/': 'start-here/consolidated/' }, anchors);
  assert.ok(chain.some((problem) => /shadows a published route/.test(problem)));
  assert.match(validateRedirects({ '/docs/a/': 'docs/b/', '/docs/b/': '' }, new Map([...anchors, ['docs/b/', new Set()]]))[0], /another redirect/);
});

test('sources shadowing a canonical route, non-local and malformed sources are rejected', () => {
  assert.match(validateRedirects({ '/start-here/consolidated/': '' }, anchors)[0], /shadows a published route/);
  assert.match(validateRedirects({ '/': '#section' }, anchors)[0], /shadows a published route/);
  for (const source of ['docs/a/', '/docs/a', '/docs/a/?x=1', '/docs/a/#x', '//evil.test/', '/docs/../a/', '/https:/x/', '/docs/%2e%2e/a/', '/docs/..\\a/', '/\\evil.test/', '/docs/%E0%A4%A/']) {
    assert.ok(validateRedirects({ [source]: '' }, anchors).length > 0, source);
  }
  assert.match(validateRedirects({ '/docs/a/': 'start-here/consolidated/?x=1' }, anchors)[0], /query/);
  assert.match(validateRedirects({ '/docs/a/': 'bad%E0%A4%A/' }, anchors)[0], /malformed link/);
  assert.match(validateRedirects({ '/docs/a/': 5 }, anchors)[0], /non-string/);
});

test('the checked-in manifest validates against real routes and headings', () => {
  const manifest = JSON.parse(fs.readFileSync(new URL('../redirects.json', import.meta.url), 'utf8'));
  const docsRoot = new URL('../../docs/', import.meta.url).pathname;
  const byRoute = new Map(
    listPublished(docsRoot).map((file) => [routeFor(file), headingAnchors(fs.readFileSync(path.join(docsRoot, file), 'utf8'))]),
  );
  assert.deepEqual(validateRedirects(manifest, byRoute), []);
});
