import { defineConfig } from '@playwright/test';

const external = process.env.DOCS_BASE_URL;
const port = 4322;
const baseURL = external ?? `http://127.0.0.1:${port}/gh-app-auth/`;
const channel = process.env.PW_CHANNEL || undefined;

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.spec.ts',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL,
    channel,
    viewport: { width: 1440, height: 1000 },
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
  webServer: external
    ? undefined
    : {
        command: `npm run build && npx astro preview --host 127.0.0.1 --port ${port}`,
        url: baseURL,
        reuseExistingServer: false,
        timeout: 240_000,
      },
});
