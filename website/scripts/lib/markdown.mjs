import GithubSlugger from 'github-slugger';

const FENCE = /^( {0,3})(`{3,}|~{3,})(.*)$/;

function withoutFences(text) {
  const kept = [];
  let open = null;
  text.split(/\r?\n/).forEach((line, index) => {
    const number = index + 1;
    const match = FENCE.exec(line);
    if (open === null) {
      if (match && !(match[2][0] === '`' && match[3].includes('`'))) {
        open = { char: match[2][0], length: match[2].length };
        kept.push({ line: '', number });
      } else {
        kept.push({ line, number });
      }
      return;
    }
    if (match && match[2][0] === open.char && match[2].length >= open.length && /^[ \t]*$/.test(match[3])) open = null;
    kept.push({ line: '', number });
  });
  return kept;
}

function stripFrontmatter(text) {
  if (!text.startsWith('---\n')) return text;
  const end = text.indexOf('\n---', 4);
  if (end === -1) return text;
  const removed = text.slice(0, end + 4).split('\n').length - 1;
  return '\n'.repeat(removed) + text.slice(end + 4);
}

export function extractLinks(text) {
  const links = [];
  for (const { line, number } of withoutFences(stripFrontmatter(text))) {
    const clean = line.replace(/`[^`]*`/g, '');
    for (const match of clean.matchAll(/!?\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)/g)) {
      links.push({ href: match[1], line: number });
    }
    const definition = clean.match(/^\[[^\]]+\]:\s*(\S+)/);
    if (definition) links.push({ href: definition[1], line: number });
  }
  return links;
}

function headingText(raw) {
  return raw
    .replace(/\s+#+\s*$/, '')
    .replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/[`*_]/g, '')
    .replace(/<[^>]+>/g, '')
    .trim();
}

export function headingAnchors(text) {
  const slugger = new GithubSlugger();
  const anchors = new Set();
  for (const { line } of withoutFences(stripFrontmatter(text))) {
    const heading = line.match(/^#{1,6}\s+(.*)$/);
    if (heading) anchors.add(slugger.slug(headingText(heading[1])));
    for (const explicit of line.matchAll(/<a\s+(?:name|id)="([^"]+)"/g)) anchors.add(explicit[1]);
  }
  return anchors;
}

export function frontmatterOf(text) {
  if (!text.startsWith('---\n')) return {};
  const end = text.indexOf('\n---', 4);
  if (end === -1) return {};
  const fields = {};
  for (const line of text.slice(4, end).split('\n')) {
    const match = line.match(/^([A-Za-z]+):\s*(.*)$/);
    if (match) fields[match[1]] = match[2];
  }
  return fields;
}
