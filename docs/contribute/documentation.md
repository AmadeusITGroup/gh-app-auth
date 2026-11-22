---
title: Documentation guide
description: How the documentation is written, built, tested and published, and how to maintain the generated parts.
sidebar:
  order: 7
---

The documentation lives in `docs/` and is built into a static site by the Astro Starlight project in
`website/`. There is one copy of the prose: the Markdown in `docs/`.

## Layout

| Path | Contents |
| --- | --- |
| `docs/index.mdx` | The home page |
| `docs/start-here`, `guides`, `reference`, `understand`, `troubleshoot`, `contribute` | The published sections. Folders and files use lowercase-kebab names |
| `docs/reference/cli/` | Generated command pages, plus the hand-written `index.md` |
| `docs/README.md` | Short directory index for readers browsing on GitHub. It is not published |
| `docs/adr/` | The original immutable ADRs and their index. They are linked, not copied or published |
| `website/` | Site shell: configuration, components, styles, scripts and browser tests |

Only the six section folders and the home page are published; the rest of `docs/` (the directory index
and the ADRs) is not.

## Writing pages

- Each page needs a `title` and a `description` in its frontmatter. Add the page to `website/sidebar.mjs`;
  the source check fails for pages missing from the sidebar.
- Write links as **relative source paths to real files**, such as
  `[routing](../understand/credential-routing.mdx)`. The build rewrites links to published pages into
  site routes, and links to anything else in the repository into canonical GitHub source URLs. A link to
  a missing file fails the build. Do not use site URLs or root-absolute paths.
- Do not start a page with a `# Title`; the frontmatter title renders as the single page heading.
- Do not use task-list checkboxes (`- [ ]`); they render as unlabeled form controls and fail the accessibility tests. Use ordinary lists.
- Use placeholder values, never real credentials, tokens or key material.
- Maintain documentation alongside the source; keep general guidance version-neutral.
  Mention versions only for behavior introduced, changed or removed in a particular release,
  historical records or intentional pins. Mark unreleased behavior explicitly until it ships.
- Diagrams are accessible HTML components with directed, labelled relationships and a text
  equivalent. Do not add a diagram renderer dependency.

## Generated and checked parts

All commands run in `website/`:

| Command | What it does |
| --- | --- |
| `npm run reference:generate` | Builds the CLI once in a temporary directory and rewrites the CLI pages from `--help` in a clean temporary `HOME`, with no user credentials or debug variables |
| `npm run reference:check` | Fails if a CLI page is stale or missing |
| `npm run check` | Type check, Markdown links and anchors, sidebar coverage, redirect targets, CLI reference drift, composite-action input and output names and nonempty defaults against `action.yml`, configuration examples and unit tests |
| `npm run build` | Build the site and the Pagefind search index into `website/dist` |
| `npm run test:e2e` | Build, serve on port 4322 and run the Playwright browser tests |

`npm run reference:generate` does not remove pages for obsolete commands. Retiring a command page needs a manual review of the page, the sidebar and the redirect policy; do not delete it automatically or leave a legacy stub.

`npm run build` and `npm run dev` need only Node.js and the checked-in files. `npm run check` also needs Go,
because the CLI reference and the configuration examples are verified against the real code.

The configuration check is a small Go program, `website/scripts/checkconfig`. It decodes every
configuration sample in the [configuration reference](../reference/configuration.md) strictly and
validates it, and requires App, PAT and mixed samples. Every top-level `yaml` or `yml` fenced block on that page (backticks or tildes, at least three markers) must be one complete configuration document with strict field validation; use `text` for fragments. An unterminated block or a second document fails the check, and blocks tagged `yaml invalid` must be rejected.

## Browser tests

The Playwright tests cover every published page: navigation, the mobile menu, search, the copy button,
theme persistence, the custom 404 page, heading anchors, redirects, horizontal overflow at 320 and 390
pixels and axe checks against WCAG 2.0, 2.1 and 2.2 A and AA in light and dark themes. They do not use
real credentials or provider APIs and are not end-to-end tests of gh-app-auth itself.

Locally, set `PW_CHANNEL=chrome` to use an installed Chrome. In CI the default Playwright Chromium is
used. The managed server runs on port 4322 and is never reused. Set `DOCS_BASE_URL` to test a
server you started yourself.

## Redirects

`website/redirects.json` is a flat map from a former site path to a canonical route, for example
`"/docs/installation/": "start-here/first-clone/#install"`.

- A source starts and ends with `/` and is a local path: no query, fragment, traversal or backslashes, and it
  must not shadow a real published route.
- A target is relative to the site base, with `""` meaning the home page, and may carry a fragment
  (`#anchor`). It must be an actual canonical route, not another redirect, and the fragment must be a real
  heading on that page. `npm run check` validates this against the published routes and their parsed
  headings, and a browser test follows every entry.
- The build generates a static HTML redirect page for each entry. There are no Markdown stub files, and
  old external GitHub `blob` links cannot be redirected.

Add an entry when you rename or move a published page.

## Publishing

The `Documentation` workflow (`.github/workflows/docs.yml`) runs the checks, the build and the browser
tests on pull requests and pushes. It uploads the site to GitHub Pages only on a push to `main` in
the canonical repository, or on a manual run on `main` with the publish option. Pull requests never
publish. Forks publish only by deliberate manual opt-in on their own `main`.

## Maintaining the vendored search component

`website/src/components/Search.astro` is a copy of the Search component from `@astrojs/starlight`
0.42.11. It differs from upstream in four ways: it imports the icon from `@astrojs/starlight/components`,
the button's accessible name comes from its visible label with the shortcut hidden from assistive
technology, the label is visually hidden on small screens so the icon button keeps a name, and opening the
dialog explicitly initializes Pagefind through one idempotent function, so the idle-time preload is only an
optimization and focus moves to the input only if the dialog is still open. Upstream code and comments are
otherwise unchanged.

Starlight is MIT licensed. The license text is kept verbatim in
`website/LICENSES/starlight-search.txt`. When you upgrade Starlight, re-diff the new upstream
`Search.astro` against this copy, reapply the four changes and check that the accessibility tests
still pass, or drop the copy if upstream has fixed the name.

## Dependencies

Versions in `website/package.json` are pinned exactly and installed with `npm ci` from
`package-lock.json`. Prefer releases at least a week old, and read the peer dependency notes before
upgrading Astro or Starlight. The workflow's action versions are pinned and verified through
`.github/workflows/actions.lock`.
