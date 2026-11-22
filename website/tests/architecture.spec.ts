import { expect, test, type Locator } from '@playwright/test';

const outDir = process.env.DOCS_SCREENSHOT_DIR;

async function renderedSvgFontSize(svg: Locator) {
  return svg.evaluate((element) => {
    const box = element as unknown as SVGSVGElement;
    const fontSize = parseFloat(getComputedStyle(box).fontSize);
    return fontSize * (box.getBoundingClientRect().width / box.viewBox.baseVal.width);
  });
}

test.describe('architecture C1 diagram', () => {
  test('desktop shows a readable SVG graph with a text version', async ({ page }) => {
    await page.goto('understand/architecture/');
    const svg = page.locator('.c1 svg[role="img"]');
    await expect(svg).toBeVisible();
    await expect(svg).toHaveAccessibleName('C1 system context of gh-app-auth');
    const labels = await svg.locator('.node text').evaluateAll((elements) =>
      elements.map((element) => {
        const spans = [...element.querySelectorAll('tspan')];
        const text = spans.length ? spans.map((span) => span.textContent ?? '').join(' ') : element.textContent ?? '';
        return text.replace(/\s+/g, ' ').trim();
      }),
    );
    expect(labels).toEqual(['Developer or CI job', 'Git and GitHub CLI', 'gh-app-auth', 'GitHub.com / GHES', 'Other HTTPS Git hosts']);
    await expect(svg.locator('.edge')).toHaveCount(5);
    expect(await renderedSvgFontSize(svg)).toBeGreaterThanOrEqual(13);
    const details = page.locator('.c1 details.text-version');
    await expect(details).toBeVisible();
    await details.locator('summary').click();
    await expect(details.locator('li')).toHaveCount(5);
    await expect(page.locator('.c1 .cards')).toBeHidden();
  });

  test('desktop does not overflow horizontally', async ({ page }) => {
    await page.goto('understand/architecture/');
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(0);
  });

  for (const width of [820, 1024, 1280, 1440]) {
    test(`at ${width}px the graph is readable or the cards take over`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto('understand/architecture/');
      const svg = page.locator('.c1 svg[role="img"]');
      if (await svg.isVisible()) {
        expect(await renderedSvgFontSize(svg)).toBeGreaterThanOrEqual(13);
        await expect(page.locator('.c1 .cards')).toBeHidden();
      } else {
        await expect(page.locator('.c1 .cards li')).toHaveCount(5);
        await expect(page.locator('.c1 .cards li').first()).toBeVisible();
      }
      if (width === 1440) await expect(svg).toBeVisible();
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
      expect(overflow).toBeLessThanOrEqual(0);
    });
  }

  for (const width of [390, 320]) {
    test.describe(`${width}px`, () => {
      test.use({ viewport: { width, height: 844 } });

      test('shows the cards without overflow', async ({ page }) => {
        await page.goto('understand/architecture/');
        await expect(page.locator('.c1 svg[role="img"]')).toBeHidden();
        await expect(page.locator('.c1 .cards li')).toHaveCount(5);
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
        expect(overflow).toBeLessThanOrEqual(0);
      });
    });
  }

  test.describe('mobile', () => {
    test.use({ viewport: { width: 390, height: 844 } });

    test('hides the SVG and shows the relationship cards', async ({ page }) => {
      await page.goto('understand/architecture/');
      await expect(page.locator('.c1 .graph')).toBeHidden();
      await expect(page.locator('.c1 .cards li')).toHaveCount(5);
      await expect(page.locator('.c1 .cards li').first()).toBeVisible();
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
      expect(overflow).toBeLessThanOrEqual(0);
    });
  });
});

test.describe('architecture screenshots', () => {
  test.skip(!outDir, 'set DOCS_SCREENSHOT_DIR to capture screenshots');

  for (const viewport of [
    { name: 'desktop', width: 1440, height: 1000 },
    { name: 'desktop1280', width: 1280, height: 900 },
    { name: 'mobile', width: 390, height: 844 },
  ]) {
    for (const theme of ['light', 'dark']) {
      test(`${viewport.name} ${theme} architecture sections`, async ({ page }) => {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('understand/architecture/');
        await page.evaluate((value) => localStorage.setItem('starlight-theme', value), theme);
        await page.reload();
        const base = `${outDir}/${viewport.name}-${theme}-architecture`;
        await page.locator('.c1').scrollIntoViewIfNeeded();
        await page.locator('.c1').screenshot({ path: `${base}-c1.png` });
        for (const id of ['c2-containers-relationship-view', 'c3-core-components-relationship-view']) {
          await page.locator(`#${id}`).scrollIntoViewIfNeeded();
          await page.screenshot({ path: `${base}-${id.slice(0, 2)}-scroll.png` });
        }
      });
    }
  }
});
