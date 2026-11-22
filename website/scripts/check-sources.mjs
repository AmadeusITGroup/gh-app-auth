import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { listPublished, routeFor } from '../src/lib/published.mjs';
import YAML from 'yaml';
import { validateRedirects } from '../src/lib/redirects.mjs';
import { checkMarkdownLinks } from './lib/check-links.mjs';
import { headingAnchors } from './lib/markdown.mjs';
import { sidebar } from '../sidebar.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const docsRoot = path.join(repoRoot, 'docs');
const problems = [];
const report = (file, message) => problems.push(`${path.relative(repoRoot, file)}: ${message}`);

const published = listPublished(docsRoot);
const publishedRoutes = new Set(published.map(routeFor));

const extraSources = [
  'README.md',
  'CONTRIBUTING.md',
  'AGENTS.md',
  'MAINTAINERS.md',
  '.github/actions/README.md',
  'test/e2e/README.md',
  'test/e2e/scripts/README.md',
  ...fs.readdirSync(docsRoot).filter((name) => name.endsWith('.md')).map((name) => `docs/${name}`),
  ...fs.readdirSync(path.join(docsRoot, 'adr')).map((name) => `docs/adr/${name}`),
];
const sources = [...new Set([...published.map((name) => `docs/${name}`), ...extraSources])].filter((name) =>
  fs.existsSync(path.join(repoRoot, name)),
);

const anchorCache = new Map();
function anchorsOf(file) {
  if (!anchorCache.has(file)) anchorCache.set(file, headingAnchors(fs.readFileSync(file, 'utf8')));
  return anchorCache.get(file);
}

for (const relative of sources) {
  const file = path.join(repoRoot, relative);
  for (const problem of checkMarkdownLinks(file, fs.readFileSync(file, 'utf8'), anchorsOf)) report(file, problem);
}

for (const name of published) {
  const file = path.join(docsRoot, name);
  const text = fs.readFileSync(file, 'utf8');
  const match = text.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n/);
  if (!match) {
    report(file, 'missing frontmatter');
    continue;
  }
  let meta;
  try {
    meta = YAML.parse(match[1]);
  } catch (error) {
    report(file, `invalid frontmatter YAML: ${error.message.split('\n')[0]}`);
    continue;
  }
  if (!meta.title) report(file, 'missing frontmatter title');
  if (!meta.description) report(file, 'missing frontmatter description');
  if (/^#\s/m.test(text.slice(match[0].length).replace(/```[\s\S]*?```/g, ''))) {
    report(file, 'page body starts a top-level heading; the frontmatter title is the page heading');
  }
  for (const action of meta.hero?.actions ?? []) {
    if (!publishedRoutes.has(String(action.link).replace(/^\//, ''))) report(file, `hero link ${action.link} is not a published route`);
  }
}

const slugs = [];
(function collect(items) {
  for (const item of items) {
    if (item.slug) slugs.push(item.slug);
    if (item.items) collect(item.items);
  }
})(sidebar);
const knownSlugs = new Set(published.map((name) => routeFor(name).replace(/\/$/, '')));
for (const slug of slugs) if (!knownSlugs.has(slug)) problems.push(`sidebar slug has no page: ${slug}`);
for (const slug of knownSlugs) {
  if (slug !== '' && !slugs.includes(slug)) problems.push(`published page missing from sidebar: ${slug}`);
}

const redirects = JSON.parse(fs.readFileSync(new URL('../redirects.json', import.meta.url), 'utf8'));
const anchorsByRoute = new Map(
  published.map((name) => [routeFor(name), headingAnchors(fs.readFileSync(path.join(docsRoot, name), 'utf8'))]),
);
problems.push(...validateRedirects(redirects, anchorsByRoute));

if (problems.length > 0) {
  console.error(`Documentation source check failed:\n- ${problems.join('\n- ')}`);
  process.exit(1);
}
console.log(`Checked ${sources.length} Markdown sources, ${published.length} published pages, ${slugs.length} sidebar entries`);
