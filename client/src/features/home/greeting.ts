import { useEffect, useState } from 'react'

// The hours each greeting starts at, in the reader's own time. Before 5 AM
// it is still the evening for someone up late, not the morning.
const morning = 5
const afternoon = 12
const evening = 17

export function greeting(now: Date): string {
  const hour = now.getHours()
  if (hour >= morning && hour < afternoon) return 'Good morning'
  if (hour >= afternoon && hour < evening) return 'Good afternoon'
  return 'Good evening'
}

// nextGreetingChange is when greeting() next says something different.
export function nextGreetingChange(now: Date): Date {
  const next = new Date(now)
  next.setMinutes(0, 0, 0)
  const hour = now.getHours()
  if (hour < morning) next.setHours(morning)
  else if (hour < afternoon) next.setHours(afternoon)
  else if (hour < evening) next.setHours(evening)
  else {
    next.setDate(next.getDate() + 1)
    next.setHours(morning)
  }
  return next
}

// useHomeClock is the time Home shows. Home reads the clock on every render;
// this also re-renders it when the greeting turns over and at midnight, when
// the date line changes, so a Home left open doesn't go stale.
export function useHomeClock(): Date {
  const [turn, setTurn] = useState(0)
  useEffect(() => {
    const now = new Date()
    const midnight = new Date(now)
    midnight.setHours(24, 0, 0, 0)
    const next = Math.min(nextGreetingChange(now).getTime(), midnight.getTime())
    const timer = window.setTimeout(() => setTurn((t) => t + 1), next - now.getTime() + 1000)
    return () => window.clearTimeout(timer)
  }, [turn])
  return new Date()
}
