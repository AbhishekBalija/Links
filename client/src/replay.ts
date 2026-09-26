import { addIntegration, replayIntegration } from '@sentry/react'

// Session Replay is the heaviest part of Sentry, so it loads after the app
// has started instead of holding up the first screen.
export function startReplay() {
  addIntegration(replayIntegration({ maskAllText: true, blockAllMedia: true }))
}
