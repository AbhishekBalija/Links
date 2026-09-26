import * as Sentry from "@sentry/react";

Sentry.init({
  dsn: import.meta.env.VITE_SENTRY_DSN,
  environment: import.meta.env.MODE,
  integrations: [Sentry.browserTracingIntegration()],
  tracesSampleRate: 1.0,
  tracePropagationTargets: [window.location.origin],
  replaysSessionSampleRate: 0.1,
  replaysOnErrorSampleRate: 1.0,
});

// Replay joins once the browser is idle; errors before then are still
// reported, just without a replay.
const loadReplay = () => import("./replay").then(({ startReplay }) => startReplay());
if ("requestIdleCallback" in window) window.requestIdleCallback(loadReplay);
else setTimeout(loadReplay, 2000);
