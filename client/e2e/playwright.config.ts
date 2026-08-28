import { defineConfig } from '@playwright/test'
import { fileURLToPath } from 'url'

const clientRoot = fileURLToPath(new URL('..', import.meta.url))

export default defineConfig({
  testDir: './scenarios',
  timeout: 60000,
  expect: { timeout: 10000 },
  fullyParallel: false,
  retries: 0,
	workers: 1,
	outputDir: `${clientRoot}/test-results`,

  globalSetup: new URL('globalSetup.ts', import.meta.url).pathname,
  globalTeardown: new URL('globalTeardown.ts', import.meta.url).pathname,

  use: {
    baseURL: 'http://localhost:5174',
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
})
