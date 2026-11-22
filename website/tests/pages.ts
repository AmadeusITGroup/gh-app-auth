import { fileURLToPath } from 'node:url';
import { listPublished, routeFor } from '../src/lib/published.mjs';

const docsRoot = fileURLToPath(new URL('../../docs/', import.meta.url));

export const pages: string[] = listPublished(docsRoot).map(routeFor);

export const slugOf = (path: string) => path.replace(/\/$/, '').replace(/\//g, '-') || 'home';

export const primaryRoutes = [
  'start-here/first-clone/',
  'guides/ci/',
  'reference/cli/',
  'understand/credential-routing/',
  'troubleshoot/',
  'contribute/development/',
];
