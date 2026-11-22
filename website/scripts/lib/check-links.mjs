import fs from 'node:fs';
import path from 'node:path';
import { isExternalHref, splitHref } from '../../src/lib/links.mjs';
import { extractLinks, headingAnchors } from './markdown.mjs';

export function checkMarkdownLinks(file, text, anchorsOf = (target) => headingAnchors(fs.readFileSync(target, 'utf8'))) {
  const problems = [];
  for (const { href, line } of extractLinks(text)) {
    if (!href.startsWith('#') && isExternalHref(href)) continue;
    let parts;
    try {
      parts = splitHref(href);
    } catch (error) {
      problems.push(`line ${line}: ${error.message}`);
      continue;
    }
    const target = parts.rawPath === '' ? file : path.resolve(path.dirname(file), parts.decodedPath);
    if (!fs.existsSync(target)) {
      problems.push(`line ${line}: missing target ${href}`);
      continue;
    }
    if (parts.anchor && /\.mdx?$/.test(target) && !anchorsOf(target).has(parts.anchor)) {
      problems.push(`line ${line}: missing anchor ${href}`);
    }
  }
  return problems;
}
