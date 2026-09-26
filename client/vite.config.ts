import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import { sentryVitePlugin } from '@sentry/vite-plugin'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  // Vercel sets VERCEL=1 on its builds; a local `bun run build` is also
  // production mode and would otherwise upload with the token in .env.local.
  const uploadSourceMaps = mode === 'production' && process.env.VERCEL === '1' && Boolean(env.SENTRY_AUTH_TOKEN)

  return {
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    // Source maps are only built and sent to Sentry from Vercel's production
    // builds, never from a laptop or CI.
    build: { sourcemap: uploadSourceMaps ? 'hidden' : false },
    plugins: [
      react(),
      tailwindcss(),
      uploadSourceMaps &&
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
