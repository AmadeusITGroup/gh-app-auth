import fs from 'node:fs';
import path from 'node:path';
import YAML from 'yaml';

export const sectionDirs = [
  'start-here',
  'guides',
  'reference',
  'understand',
  'troubleshoot',
  'contribute',
];

export const publishedPatterns = [
  'index.mdx',
  `{${sectionDirs.join(',')}}/**/*.{md,mdx}`,
];

const sourceExtension = /\.mdx?$/;

function walk(directory, found) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const full = path.join(directory, entry.name);
    if (entry.isDirectory()) walk(full, found);
    else if (sourceExtension.test(entry.name)) found.push(full);
  }
}

export function isDraft(file) {
  const text = fs.readFileSync(file, 'utf8');
  const match = text.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);
  if (match === null) return false;
  let meta;
  try {
    meta = YAML.parse(match[1]);
  } catch (error) {
    throw new Error(`${file}: invalid frontmatter YAML: ${error.message.split('\n')[0]}`);
  }
  return meta !== null && typeof meta === 'object' && meta.draft === true;
}

export function listPublished(docsRoot) {
  const found = [];
  if (fs.existsSync(path.join(docsRoot, 'index.mdx'))) found.push(path.join(docsRoot, 'index.mdx'));
  for (const section of sectionDirs) {
    const directory = path.join(docsRoot, section);
    if (fs.existsSync(directory)) walk(directory, found);
  }
  return found
    .filter((file) => !isDraft(file))
    .map((file) => path.relative(docsRoot, file).split(path.sep).join('/'))
    .sort();
}

export function routeFor(relativePath) {
  const withoutExtension = relativePath.replace(sourceExtension, '');
  if (withoutExtension === 'index') return '';
  if (withoutExtension.endsWith('/index')) return `${withoutExtension.slice(0, -'index'.length)}`;
  return `${withoutExtension}/`;
}
