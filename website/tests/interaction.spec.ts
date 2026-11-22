import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Page } from '@playwright/test';

test.describe('search', () => {
  test('keyboard search finds routing and the result opens, with back navigation', async ({ page }) => {
    await page.goto('');
    await page.keyboard.press('Control+k');
    const dialog = page.locator('dialog[open]');
    await expect(dialog).toBeVisible();
    const input = dialog.locator('.pagefind-ui__search-input');
    await expect(input).toBeAttached();
    await page.keyboard.press('Escape');
    await expect(dialog).toHaveCount(0);
    await page.keyboard.press('Control+k');
    await expect(input).toBeFocused();
    await input.fill('routing');
    const result = dialog.locator('.pagefind-ui__result-link').first();
    await expect(result).toBeVisible();
    await result.click();
    await expect(page).toHaveURL(/credential-routing\//);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await page.goBack();
    await expect(page).toHaveURL(/\/gh-app-auth\/$/);
  });

  test('Escape closes search and returns focus to the search button', async ({ page }) => {
    await page.goto('');
    const button = page.locator('button[data-open-modal]');
    await expect(button).toBeEnabled();
    await button.click();
    await expect(page.locator('dialog[open]')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(button).toBeFocused();
  });

  test('search button has a name that includes its visible label', async ({ page }) => {
    await page.goto('');
    const button = page.getByRole('button', { name: 'Search' });
    await expect(button).toHaveCount(1);
    await expect(button).toHaveAttribute('aria-keyshortcuts', /Control\+K|Meta\+K/);
  });
});

test.describe('search finds consolidated pages', () => {
  for (const [query, route] of [
    ['personal access tokens', '/guides/credentials/'],
    ['diagnostics', '/troubleshoot/'],
  ]) {
    test(`searching "${query}" finds and opens ${route}`, async ({ page }) => {
      await page.goto('');
      await page.keyboard.press('Control+k');
      const dialog = page.locator('dialog[open]');
      await expect(dialog).toBeVisible();
      const input = dialog.locator('.pagefind-ui__search-input');
      await expect(input).toBeAttached();
      await page.keyboard.press('Escape');
      await page.keyboard.press('Control+k');
      await input.fill(query);
      const match = dialog.locator(`.pagefind-ui__result-link[href$="${route}"], .pagefind-ui__result-link[href*="${route}#"]`).first();
      await expect(match).toBeVisible();
      await match.click();
      await expect(page).toHaveURL(new RegExp(`${route.replace(/\//g, '\\/')}`));
      await expect(page.locator('h1')).toHaveCount(1);
    });
  }
});

const holdIdleWork = (page: Page) =>
  page.addInitScript(() => {
    window.requestIdleCallback = () => 0;
    window.cancelIdleCallback = () => undefined;
  });

async function openSearch(page: Page, method: 'keyboard' | 'pointer') {
  const button = page.locator('button[data-open-modal]');
  await expect(button).toBeEnabled();
  if (method === 'keyboard') await page.keyboard.press('Control+k');
  else await button.click();
  const dialog = page.locator('dialog[open]');
  await expect(dialog).toBeVisible();
  return dialog;
}

test.describe('search initializes on first use without idle work', () => {
  for (const width of [1440, 390]) {
    for (const method of ['keyboard', 'pointer'] as const) {
      test(`${method} open at ${width}px focuses a single input and finds routing`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 });
        await holdIdleWork(page);
        await page.goto('');
        const dialog = await openSearch(page, method);
        const input = dialog.locator('.pagefind-ui__search-input');
        await expect(input).toHaveCount(1);
        await expect(input).toBeFocused();
        if (method === 'keyboard' && width === 1440) {
          await page.keyboard.press('Escape');
          await expect(page.locator('dialog[open]')).toHaveCount(0);
          await openSearch(page, 'keyboard');
          await expect(page.locator('.pagefind-ui__search-input')).toHaveCount(1);
          await expect(page.locator('.pagefind-ui__search-input')).toBeFocused();
        }
        await input.fill('routing');
        const result = dialog.locator('.pagefind-ui__result-link').first();
        await expect(result).toBeVisible();
        await result.click();
        await expect(page).toHaveURL(/credential-routing\//);
      });
    }
  }

  for (const width of [1440, 390]) {
    for (const theme of ['light', 'dark']) {
      test(`the open search dialog has no axe violations at ${width}px in ${theme}`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 });
        await holdIdleWork(page);
        await page.addInitScript((value) => localStorage.setItem('starlight-theme', value), theme);
        await page.goto('');
        const dialog = await openSearch(page, 'pointer');
        await expect(dialog.locator('.pagefind-ui__search-input')).toBeFocused();
        const results = await new AxeBuilder({ page })
          .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
          .analyze();
        expect(results.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)).toEqual([]);
        await page.screenshot({ path: `${process.env.DOCS_SCREENSHOT_DIR ?? test.info().outputDir}/search-open-${width}-${theme}.png` });
      });
    }
  }

  test('closing while the search UI loads does not reopen the dialog or steal focus', async ({ page }) => {
    await holdIdleWork(page);
    let release: () => void = () => undefined;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/_astro/ui-core.*.js', async (route) => {
      await gate;
      await route.continue();
    });
    await page.goto('');
    const button = page.locator('button[data-open-modal]');
    const pending = page.waitForRequest('**/_astro/ui-core.*.js');
    await openSearch(page, 'pointer');
    await pending;
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(page.locator('body[data-search-modal-open]')).toHaveCount(0);
    release();
    const input = page.locator('dialog .pagefind-ui__search-input');
    await expect(input).toHaveCount(1);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(input).not.toBeFocused();
    await expect(button).toBeFocused();
    await expect(page.locator('body[data-search-modal-open]')).toHaveCount(0);
    await openSearch(page, 'keyboard');
    await expect(page.locator('.pagefind-ui__search-input')).toHaveCount(1);
    await expect(page.locator('.pagefind-ui__search-input')).toBeFocused();
  });
});

test.describe('search on mobile', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('icon button keeps its accessible name', async ({ page }) => {
    await page.goto('');
    await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
  });
});

test('copy button copies the code block', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('start-here/first-clone/');
  const block = page.locator('.expressive-code').first();
  await block.hover();
  await block.locator('.copy button').click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    'gh extension install AmadeusITGroup/gh-app-auth\n'.trimEnd(),
  );
});

test('theme choice persists across navigation', async ({ page }) => {
  await page.goto('');
  const select = page.locator('starlight-theme-select select:visible');
  await select.selectOption('dark');
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.getByRole('link', { name: 'Make your first clone' }).click();
  await expect(page).toHaveURL(/start-here\/first-clone\/$/);
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.locator('#starlight__sidebar a[href$="/understand/credential-routing/"]').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await select.selectOption('light');
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
});
