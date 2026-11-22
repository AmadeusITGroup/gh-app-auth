import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';
import { pages } from './pages';

for (const width of [390, 320]) {
  test.describe(`${width}px viewport`, () => {
    test.use({ viewport: { width, height: 800 } });
    for (const path of pages) {
      test(`no horizontal overflow on ${path || 'home'}`, async ({ page }) => {
        await page.goto(path);
        const overflow = await page.evaluate(
          () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
        );
        expect(overflow).toBeLessThanOrEqual(0);
      });
    }
  });
}

for (const theme of ['light', 'dark'] as const) {
  for (const path of [...pages, 'does-not-exist/']) {
    test(`axe wcag a/aa/2.1/2.2 ${theme} ${path || 'home'}`, async ({ page }) => {
      await page.goto(path);
      await page.evaluate((value) => {
        localStorage.setItem('starlight-theme', value);
      }, theme);
      await page.reload();
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
        .analyze();
      expect(
        results.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`),
      ).toEqual([]);
    });
  }
}

test('mobile whole-page axe passes with the menu closed', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
    .analyze();
  expect(results.violations.map((v) => v.id)).toEqual([]);
});

test('no console errors or failed requests while browsing', async ({ page }) => {
  const problems: string[] = [];
  page.on('console', (m) => {
    if (m.type() === 'error' || m.type() === 'warning') problems.push(`${m.type()}: ${m.text()}`);
  });
  page.on('pageerror', (e) => problems.push(`pageerror: ${e.message}`));
  page.on('response', (r) => {
    if (r.status() >= 400) problems.push(`${r.status()} ${r.url()}`);
  });
  for (const path of pages) await page.goto(path);
  expect(problems).toEqual([]);
});
