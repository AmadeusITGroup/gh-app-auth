import fs from 'node:fs';
import path from 'node:path';

const slugs = (...names) => names.map((slug) => ({ slug }));

const cliDirectory = new URL('../docs/reference/cli/', import.meta.url);
const cliCommands = fs
  .readdirSync(cliDirectory)
  .filter((name) => name.endsWith('.md') && name !== 'index.md')
  .map((name) => path.basename(name, '.md'))
  .sort();

export const sidebar = [
  { label: 'Start here', items: slugs('start-here/first-clone') },
  {
    label: 'Guides',
    items: slugs('guides/credentials', 'guides/git-credential-helper', 'guides/ci'),
  },
  {
    label: 'Reference',
    items: [
      {
        label: 'CLI commands',
        collapsed: true,
        items: [{ slug: 'reference/cli' }, ...cliCommands.map((name) => ({ slug: `reference/cli/${name}` }))],
      },
      ...slugs('reference/configuration', 'reference/github-actions'),
    ],
  },
  {
    label: 'Understand',
    items: slugs('understand/credential-routing', 'understand/security-model', 'understand/architecture'),
  },
  { label: 'Troubleshoot', items: slugs('troubleshoot') },
  {
    label: 'Contribute',
    items: slugs(
      'contribute/development',
      'contribute/testing',
      'contribute/e2e-infrastructure',
      'contribute/releasing',
      'contribute/documentation',
    ),
  },
];
