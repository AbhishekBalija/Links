import type { Opportunity } from './types'

const DAY = 24 * 60 * 60 * 1000

// calendarDays counts midnights between now and the deadline, in local time,
// so "tomorrow" means the next date on the calendar, not 24 hours away.
function calendarDays(iso: string, now: Date): number {
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const due = new Date(iso)
  const end = new Date(due.getFullYear(), due.getMonth(), due.getDate())
  return Math.round((end.getTime() - start.getTime()) / DAY)
}

// daysLeft says how long is left to apply: "Closes today", "2 days left".
export function daysLeft(applyBy: string, now = new Date()): string {
  if (new Date(applyBy) <= now) return 'Closed'
  const days = calendarDays(applyBy, now)
  if (days <= 0) return 'Closes today'
  if (days === 1) return 'Closes tomorrow'
  return `${days} days left`
}

// isUrgent is a deadline three calendar days away or less, still ahead.
export function isUrgent(applyBy: string, now = new Date()): boolean {
  return new Date(applyBy) > now && calendarDays(applyBy, now) <= 3
}

// jobGroup is the heading an open Opportunity is listed under.
export function jobGroup(applyBy: string, now = new Date()): string {
  return calendarDays(applyBy, now) <= 7 ? 'Closing this week' : 'Later'
}

// deadlineParts is the day and month on the apply-by tile: "02", "OCT".
export function deadlineParts(iso: string): { day: string; month: string } {
  const d = new Date(iso)
  return {
    day: String(d.getDate()).padStart(2, '0'),
    month: d.toLocaleDateString('en-IN', { month: 'short' }).slice(0, 3).toUpperCase(),
  }
}

// deadlineLine is the full deadline: "Friday 2 October" and "11:59 pm".
export function deadlineLine(iso: string): { date: string; time: string } {
  const d = new Date(iso)
  return {
    date: `${d.toLocaleDateString('en-IN', { weekday: 'long' })} ${d.getDate()} ${d.toLocaleDateString('en-IN', { month: 'long' })}`,
    time: d.toLocaleTimeString('en-IN', { hour: 'numeric', minute: '2-digit', hour12: true }).toLowerCase(),
  }
}

// jobMeta is the line under a role: company, place and pay.
export function jobMeta(job: Pick<Opportunity, 'company' | 'location' | 'compensation'>): string {
  return [job.company, job.location, job.compensation].filter(Boolean).join(' · ')
}
