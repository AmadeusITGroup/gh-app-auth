import { expect, test } from '@playwright/test';
import fs from 'node:fs';
import { pages, primaryRoutes } from './pages';

const groups = ['Start here', 'Guides', 'Reference', 'Understand', 'Troubleshoot', 'Contribute'];
const redirects: Record<string, string> = JSON.parse(
  fs.readFileSync(new URL('../redirects.json', import.meta.url), 'utf8'),
);
const redirectEntries = Object.entries(redirects);

test.describe('sidebar and landmarks', () => {
  test('sidebar shows every group and a visible link for every published page', async ({ page }) => {
    await page.goto('start-here/first-clone/');
    const sidebar = page.locator('#starlight__sidebar');
    for (const group of groups) {
      await expect(sidebar.locator('summary', { hasText: group }).first()).toBeVisible();
    }
    await sidebar.locator('summary', { hasText: 'CLI commands' }).click();
    const links = sidebar.locator('a[href^="/gh-app-auth/"]:visible');
    await expect(links).toHaveCount(pages.length - 1);
    await expect(sidebar.locator('a[aria-current="page"]')).toHaveCount(1);
    await expect(sidebar.locator('a[aria-current="page"]')).toHaveText('Get started');
  });

  test('every primary route has a sidebar link', async ({ page }) => {
    await page.goto('start-here/first-clone/');
    const sidebar = page.locator('#starlight__sidebar');
    for (const route of primaryRoutes) {
      await expect(sidebar.locator(`a[href$="/${route}"]`), route).toHaveCount(1);
    }
  });

  test('nested group "CLI commands" expands and its pages open', async ({ page }) => {
    await page.goto('start-here/first-clone/');
    const sidebar = page.locator('#starlight__sidebar');
    const link = sidebar.locator('a[href$="/reference/cli/setup/"]');
    await expect(link).toBeHidden();
    await sidebar.locator('summary', { hasText: 'CLI commands' }).click();
    await expect(link).toBeVisible();
    await link.click();
    await expect(page).toHaveURL(/reference\/cli\/setup\/$/);
    await expect(page.locator('h1')).toBeVisible();
    await expect(sidebar.locator('a[aria-current="page"][href$="/reference/cli/setup/"]')).toBeVisible();
  });

  for (const section of ['github-actions', 'jenkins', 'gitlab-ci']) {
    test(`CI guide section ${section} is reachable from the table of contents`, async ({ page }) => {
      await page.goto('guides/ci/');
      await page.locator(`starlight-toc a[href="#${section}"]`).first().click();
      await expect(page).toHaveURL(new RegExp(`/guides/ci/#${section}$`));
      await expect(page.locator(`#${section}`)).toBeVisible();
    });
  }

  for (const route of ['understand/credential-routing/', 'troubleshoot/', 'contribute/releasing/']) {
    test(`clicking the sidebar link for ${route} navigates`, async ({ page }) => {
      await page.goto('start-here/first-clone/');
      await page.locator(`#starlight__sidebar a[href$="/${route}"]`).click();
      await expect(page).toHaveURL(new RegExp(`${route}$`));
      await expect(page.locator('h1')).toBeVisible();
    });
  }

  test('home page exposes a named route-choice navigation landmark', async ({ page }) => {
    await page.goto('');
    const nav = page.getByRole('navigation', { name: 'Choose where to start' });
    await expect(nav.getByRole('link')).toHaveCount(3);
  });

  for (const path of pages) {
    test(`landmarks and single h1 on ${path || 'home'}`, async ({ page }) => {
      await page.goto(path);
      await expect(page.locator('h1')).toHaveCount(1);
      await expect(page.locator('main')).toHaveCount(1);
      await expect(page.locator('header').first()).toBeVisible();
      await expect(page.getByRole('navigation').first()).toBeAttached();
    });
  }

  test('favicon loads from the site base', async ({ page, request }) => {
    await page.goto('');
    const href = await page.locator('link[rel="shortcut icon"]').first().getAttribute('href');
    expect(href).toContain('/gh-app-auth/');
    const response = await request.get(new URL(href!, page.url()).toString());
    expect(response.status()).toBe(200);
  });

  test('unknown pages render the custom 404', async ({ page }) => {
    const response = await page.goto('does-not-exist/');
    expect(response?.status()).toBe(404);
    await expect(page.locator('h1')).toHaveText('Page not found');
    await page.getByRole('link', { name: 'Your first authenticated clone' }).click();
    await expect(page).toHaveURL(/start-here\/first-clone\/$/);
  });

  test('all internal links and anchors resolve', async ({ page, request }) => {
    const targets = new Map<string, Set<string>>();
    for (const path of pages) {
      await page.goto(path);
      const hrefs = await page.locator('a[href]').evaluateAll((as) => as.map((a) => (a as HTMLAnchorElement).href));
      for (const href of hrefs) {
        const url = new URL(href);
        if (url.origin !== new URL(page.url()).origin) continue;
        const key = url.origin + url.pathname;
        if (!targets.has(key)) targets.set(key, new Set());
        if (url.hash.length > 1) targets.get(key)!.add(decodeURIComponent(url.hash.slice(1)));
      }
    }
    expect(targets.size).toBeGreaterThan(pages.length / 2);
    for (const [url, anchors] of targets) {
      expect(url, url).toContain('/gh-app-auth/');
      const response = await request.get(url);
      expect(response.status(), url).toBe(200);
      const html = await response.text();
      for (const anchor of anchors) {
        expect(html.includes(`id="${anchor}"`), `${url}#${anchor}`).toBe(true);
      }
    }
  });

  test('every published page is linked from somewhere reachable', async ({ page }) => {
    const reached = new Set<string>();
    await page.goto('start-here/first-clone/');
    const sidebarHrefs = await page
      .locator('#starlight__sidebar a[href^="/gh-app-auth/"]')
      .evaluateAll((as) => as.map((a) => (a as HTMLAnchorElement).getAttribute('href')!));
    sidebarHrefs.forEach((h) => reached.add(h));
    for (const path of pages.filter((p) => p !== '')) {
      expect(reached.has(`/gh-app-auth/${path}`), path).toBe(true);
    }
  });

  test('heading anchors link to their headings', async ({ page }) => {
    for (const path of ['start-here/first-clone/', 'understand/architecture/']) {
      await page.goto(path);
      const headings = page.locator('.sl-markdown-content h2[id]');
      expect(await headings.count()).toBeGreaterThan(3);
      const first = headings.nth(1);
      const id = await first.getAttribute('id');
      const anchor = first.locator('xpath=..').locator(`a.sl-anchor-link[href="#${id}"]`);
      await expect(anchor).toHaveCount(1);
      await anchor.click({ force: true });
      await expect(page).toHaveURL(new RegExp(`#${id}$`));
    }
  });

  test('former PAT guide reaches the registration section', async ({ page }) => {
    await page.goto('guides/personal-access-tokens/');
    await page.waitForURL((url) => url.pathname + url.hash === '/gh-app-auth/guides/credentials/#personal-access-tokens');
    const heading = page.locator('h2#personal-access-tokens');
    await expect(heading).toHaveText('Personal access tokens');
    const section = page.locator('.sl-markdown-content > div, .sl-markdown-content').first();
    const registration = page.locator('h3', { hasText: 'Register a PAT' });
    await expect(registration).toHaveCount(1);
    const order = await page.evaluate(() => {
      const h2 = document.querySelector('h2#personal-access-tokens');
      const h3 = [...document.querySelectorAll('h3')].find((h) => h.textContent?.includes('Register a PAT'));
      if (!h2 || !h3) return null;
      const following = h2.compareDocumentPosition(h3) & Node.DOCUMENT_POSITION_FOLLOWING;
      const nextH2 = [...document.querySelectorAll('h2')].find((h) => h2.compareDocumentPosition(h) & Node.DOCUMENT_POSITION_FOLLOWING);
      const before = !nextH2 || !!(h3.compareDocumentPosition(nextH2) & Node.DOCUMENT_POSITION_FOLLOWING);
      return Boolean(following) && before;
    });
    expect(order, 'Register a PAT is under the Personal access tokens h2').toBe(true);
    await expect(section).toBeVisible();
    if (process.env.DOCS_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.DOCS_SCREENSHOT_DIR}/pat-redirect.png` });
  });

  for (const [source, target] of redirectEntries) {
    test(`former route ${source} redirects to ${target || 'the home page'}`, async ({ page }) => {
      await page.goto(source.slice(1));
      const expected = `/gh-app-auth/${target}`;
      await page.waitForURL((url) => url.pathname + url.hash === expected);
      await expect(page.locator('h1')).toHaveCount(1);
      const fragment = target.includes('#') ? target.slice(target.indexOf('#') + 1) : '';
      if (fragment) await expect(page.locator(`[id="${fragment}"]`)).toHaveCount(1);
    });
  }
});

test.describe('mobile menu', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('opens, navigates and closes', async ({ page }) => {
    await page.goto('start-here/first-clone/');
    const toggle = page.locator('button[popovertarget="starlight__sidebar"]');
    const sidebar = page.locator('#starlight__sidebar');
    await expect(sidebar).toBeHidden();
    await toggle.click();
    await expect(sidebar).toBeVisible();
    await sidebar.locator('a[href$="/understand/credential-routing/"]').click();
    await expect(page).toHaveURL(/understand\/credential-routing\/$/);
    await expect(page.locator('h1')).toHaveText('How credential routing works');
    await expect(sidebar).toBeHidden();
    await expect(toggle).toBeVisible();
  });
});
