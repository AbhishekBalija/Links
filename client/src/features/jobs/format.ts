import { clockTime, collegeDate, collegeDay, monthsShort, wallClock } from '../../shared/time/college'
import type { Opportunity } from './types'

// calendarDays counts midnights between now and the deadline on the
// college's calendar (#211), so "tomorrow" means the next date there, not
// 24 hours away.
function calendarDays(iso: string, now: Date): number {
  return collegeDay(iso) - collegeDay(now)
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
  const c = wallClock(iso)
  return {
    day: String(c.day).padStart(2, '0'),
    month: monthsShort[c.month - 1].toUpperCase(),
  }
}

// deadlineLine is the full deadline: "Friday 2 October" and "11:59 pm".
export function deadlineLine(iso: string): { date: string; time: string } {
  return { date: collegeDate(iso, 'long'), time: clockTime(iso) }
}

// jobMeta is the line under a role: company, place and pay.
export function jobMeta(job: Pick<Opportunity, 'company' | 'location' | 'compensation'>): string {
  return [job.company, job.location, job.compensation].filter(Boolean).join(' · ')
}
