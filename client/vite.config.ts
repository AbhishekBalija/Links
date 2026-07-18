import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { sentryVitePlugin } from "@sentry/vite-plugin";

export default defineConfig({
  build: { sourcemap: "hidden" },
  plugins: [
    react(),
    sentryVitePlugin({
      org: "abhi-org-w5",
      project: "links-web",
      authToken: process.env.SENTRY_AUTH_TOKEN,
    }),
  ],
})
