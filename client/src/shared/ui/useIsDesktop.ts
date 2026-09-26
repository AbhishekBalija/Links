import { useSyncExternalStore } from 'react'

// Matches Tailwind's lg breakpoint, where the desktop layout starts.
const query = '(min-width: 1024px)'

function subscribe(onChange: () => void) {
  const list = window.matchMedia(query)
  list.addEventListener('change', onChange)
  return () => list.removeEventListener('change', onChange)
}

// useIsDesktop is for the few places where phone and desktop need different
// components, not just different styles (a sheet on phones, inline on desktop).
export function useIsDesktop() {
  return useSyncExternalStore(subscribe, () => window.matchMedia(query).matches, () => true)
}
