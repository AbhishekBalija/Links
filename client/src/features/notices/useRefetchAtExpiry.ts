import { useEffect } from 'react'

// Browsers cap setTimeout at about 24.8 days; a later expiry waits for the
// next render (any refetch) to schedule itself again.
const MAX_DELAY = 2_147_483_647

// useRefetchAtExpiry asks the server again when the soonest of these notices
// expires. The server decides what is visible, so an expired notice drops
// off an open screen without the client guessing.
export function useRefetchAtExpiry(expiries: (string | null)[], refetch: () => void) {
  // A string key, so a new array with the same dates doesn't reset the timer.
  const key = expiries.filter(Boolean).join(',')

  useEffect(() => {
    const now = Date.now()
    const soonest = key
      .split(',')
      .map((iso) => (iso ? new Date(iso).getTime() : Number.NaN))
      .filter((time) => time > now)
      .reduce((min, time) => Math.min(min, time), Number.POSITIVE_INFINITY)
    if (!Number.isFinite(soonest)) return
    const delay = Math.min(soonest - now + 1000, MAX_DELAY)
    const timer = setTimeout(refetch, delay)
    return () => clearTimeout(timer)
  }, [key, refetch])
}
