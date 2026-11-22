import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { unified } from '@astrojs/markdown-remark';
import { fileURLToPath } from 'node:url';
import { listPublished } from './src/lib/published.mjs';
import { loadRedirects } from './src/lib/redirects.mjs';
import focusableCodeBlocks from './src/plugins/focusable-code.mjs';
import remarkSourceLinks from './src/plugins/source-links.mjs';
import { sidebar } from './sidebar.mjs';

const base = '/gh-app-auth/';
const repoRoot = fileURLToPath(new URL('..', import.meta.url));
const docsRoot = fileURLToPath(new URL('../docs/', import.meta.url));

export default defineConfig({
  site: 'https://AmadeusITGroup.github.io',
  base,
  trailingSlash: 'always',
  redirects: loadRedirects(new URL('./redirects.json', import.meta.url), base),
  markdown: {
    processor: unified({
      remarkPlugins: [
        [
          remarkSourceLinks,
          {
            repoRoot,
            docsRoot,
            published: listPublished(docsRoot),
            base,
            repoUrl: 'https://github.com/AmadeusITGroup/gh-app-auth',
            branch: 'main',
          },
        ],
      ],
    }),
  },
  integrations: [
    starlight({
      title: 'gh-app-auth',
      description:
        'Git credential helper and GitHub CLI extension that routes repositories to GitHub Apps and personal access tokens.',
      favicon: '/favicon.svg',
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/AmadeusITGroup/gh-app-auth',
        },
      ],
      customCss: ['./src/styles/custom.css'],
      markdown: { processedDirs: ['../docs'] },
      disable404Route: true,
      expressiveCode: { plugins: [focusableCodeBlocks] },
      components: {
        Hero: './src/components/Hero.astro',
        Search: './src/components/Search.astro',
      },
      sidebar,
    }),
  ],
});
