import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

const outDir = process.env.DOCS_SCREENSHOT_DIR;

for (const width of [1440, 390]) {
  test(`overflowing code blocks are keyboard reachable before idle work at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.addInitScript(() => {
      window.requestIdleCallback = () => 0;
      window.cancelIdleCallback = () => undefined;
    });
    await page.goto('reference/cli/setup/');

    const blocks = page.locator('.expressive-code pre');
    const overflowing = await blocks.evaluateAll((elements) =>
      elements.findIndex((element) => element.scrollWidth > element.clientWidth),
    );
    expect(overflowing, 'a code block that overflows horizontally').toBeGreaterThanOrEqual(0);
    const pre = blocks.nth(overflowing);

    await expect(pre).toHaveAttribute('tabindex', '0');

    await pre.focus();
    await expect(pre).toBeFocused();
    await page.keyboard.press('Shift+Tab');
    await expect(pre).not.toBeFocused();
    await page.keyboard.press('Tab');
    await expect(pre).toBeFocused();

    await page.keyboard.press('ArrowRight');
    await expect.poll(() => pre.evaluate((element) => element.scrollLeft)).toBeGreaterThan(0);

    if (outDir) await page.screenshot({ path: `${outDir}/code-focus-${width}.png` });

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
      .analyze();
    expect(results.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)).toEqual([]);
  });
}
