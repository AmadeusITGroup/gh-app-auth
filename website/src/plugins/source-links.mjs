import fs from 'node:fs';
import path from 'node:path';
import { isExternalHref, splitHref } from '../lib/links.mjs';
import { routeFor } from '../lib/published.mjs';

const schemePattern = /^[a-z][a-z0-9+.-]*:/i;

export function rewriteHref(href, context) {
  const { filePath, repoRoot, docsRoot, published, base, repoUrl, branch } = context;
  if (isExternalHref(href)) return href;
  const { rawPath, decodedPath, query, hash } = splitHref(href);
  if (!rawPath) return href;
  if (rawPath.startsWith('/')) {
    throw new Error(`${filePath}: root-absolute link "${href}" is not allowed; use a relative source path`);
  }
  const target = path.resolve(path.dirname(filePath), decodedPath);
  const relativeToRepo = path.relative(repoRoot, target).split(path.sep).join('/');
  if (relativeToRepo.startsWith('..') || path.isAbsolute(relativeToRepo)) {
    throw new Error(`${filePath}: link "${href}" points outside the repository`);
  }
  if (!fs.existsSync(target)) {
    throw new Error(`${filePath}: link target not found for "${href}"`);
  }
  const isDirectory = fs.statSync(target).isDirectory();
  const relativeToDocs = path.relative(docsRoot, target).split(path.sep).join('/');
  const inDocs = !relativeToDocs.startsWith('..');
  if (!isDirectory && inDocs && published.has(relativeToDocs)) {
    return `${base}${routeFor(relativeToDocs)}${query}${hash}`;
  }
  if (isDirectory && inDocs) {
    for (const index of ['index.md', 'index.mdx']) {
      const candidate = relativeToDocs ? `${relativeToDocs}/${index}` : index;
      if (published.has(candidate)) return `${base}${routeFor(candidate)}${query}${hash}`;
    }
  }
  const kind = isDirectory ? 'tree' : 'blob';
  return `${repoUrl}/${kind}/${branch}/${relativeToRepo}${hash}`;
}

function visit(node, callback) {
  callback(node);
  if (Array.isArray(node.children)) for (const child of node.children) visit(child, callback);
}

export default function remarkSourceLinks(options) {
  const published = new Set(options.published);
  return (tree, file) => {
    const context = { ...options, published, filePath: file.path };
    visit(tree, (node) => {
      if (node.type === 'link' || node.type === 'definition') {
        node.url = rewriteHref(node.url, context);
      } else if (node.type === 'image' && node.url && !schemePattern.test(node.url)) {
        throw new Error(`${file.path}: relative image "${node.url}" is not supported`);
      }
    });
  };
}
