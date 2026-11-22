import fs from 'node:fs';
import { splitHref } from './links.mjs';

export function loadRedirects(file, base) {
  const manifest = JSON.parse(fs.readFileSync(file, 'utf8'));
  return Object.fromEntries(Object.entries(manifest).map(([source, target]) => [source, `${base}${target}`]));
}

function sourceProblem(source) {
  if (!source.startsWith('/') || !source.endsWith('/')) return 'must start and end with "/"';
  if (/[?#]/.test(source)) return 'must not contain a query or fragment';
  if (source.startsWith('//') || /^\/[a-z][a-z0-9+.-]*:/i.test(source)) return 'must be a local path';
  if (source.split('/').includes('..')) return 'must not contain path traversal';
  try {
    splitHref(source);
  } catch {
    return 'is malformed';
  }
  const resolved = new URL(source, 'https://docs.invalid');
  if (resolved.origin !== 'https://docs.invalid' || resolved.pathname !== source) {
    return 'must be a normalized local path';
  }
  return null;
}

export function validateRedirects(manifest, anchorsByRoute) {
  const problems = [];
  for (const [source, target] of Object.entries(manifest)) {
    const invalid = sourceProblem(source);
    if (invalid) {
      problems.push(`redirect source ${source} ${invalid}`);
      continue;
    }
    if (anchorsByRoute.has(source.slice(1))) {
      problems.push(`redirect source ${source} shadows a published route`);
    }
    if (typeof target !== 'string') {
      problems.push(`redirect ${source} has a non-string target`);
      continue;
    }
    let parts;
    try {
      parts = splitHref(target);
    } catch (error) {
      problems.push(`redirect ${source}: ${error.message}`);
      continue;
    }
    if (parts.query) {
      problems.push(`redirect ${source} target ${target} must not contain a query`);
      continue;
    }
    const anchors = anchorsByRoute.get(parts.rawPath);
    if (!anchors) {
      problems.push(`redirect ${source} targets ${target}, which is not a published route`);
      continue;
    }
    if (Object.hasOwn(manifest, `/${parts.rawPath}`)) {
      problems.push(`redirect ${source} targets another redirect (${target})`);
    }
    if (parts.anchor && !anchors.has(parts.anchor)) {
      problems.push(`redirect ${source} targets missing anchor ${target}`);
    }
  }
  return problems;
}
