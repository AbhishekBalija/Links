import path from 'path'
import { defineConfig } from 'vitest/config'

// Unit tests for pure logic (formatting, statuses, audiences). They run in
// Node without a browser; screens are covered by the Playwright suite.
export default defineConfig({
  resolve: { alias: { '@': path.resolve(__dirname, './src') } },
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'node',
  },
})
