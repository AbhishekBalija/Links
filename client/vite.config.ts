import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import { sentryVitePlugin } from '@sentry/vite-plugin'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    build: { sourcemap: 'hidden' },
    plugins: [
      react(),
      tailwindcss(),
      sentryVitePlugin({
        org: 'abhi-org-w5',
        project: 'links-web',
        authToken: env.SENTRY_AUTH_TOKEN,
      }),
    ],
    server: {
      proxy: {
        '/api': {
          target: env.VITE_API_URL || 'http://localhost:8081',
          changeOrigin: true,
        },
      },
    },
  }
})
