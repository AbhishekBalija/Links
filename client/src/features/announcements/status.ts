import { timeAgo } from '../notices/format'
import type { Authored } from './types'

export type Tone = 'neutral' | 'live' | 'danger'

export type Standing = {
  // The short status tag: "Draft", "With CS HOD", "Live", "Expired".
  tag: string
  tone: Tone
  // What the date means for this status: "saved 3 days ago", "published …".
  when: string
  // A draft or a sent-back one opens straight in the composer; the rest
  // open read-only with their actions.
  opensInComposer: boolean
  // Sent back: the author has to act (a rejected one, or a rejected edit).
  needsAuthor: boolean
  expired: boolean
}

function shortDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })
}

export function standing(item: Authored, now = new Date()): Standing {
  const expired = item.status === 'published' && item.expires_at !== null && new Date(item.expires_at) <= now
  const base = { opensInComposer: false, needsAuthor: false, expired }
  switch (item.status) {
    case 'draft':
      return { ...base, tag: 'Draft', tone: 'neutral', when: `saved ${timeAgo(item.created_at, now)}`, opensInComposer: true }
    case 'pending':
      return { ...base, tag: `With ${item.approver ?? 'approver'}`, tone: 'neutral', when: `sent ${timeAgo(item.created_at, now)}` }
    case 'rejected':
      return { ...base, tag: 'Sent back', tone: 'danger', when: 'sent back', opensInComposer: true, needsAuthor: true }
    case 'withdrawn':
      return { ...base, tag: 'Withdrawn', tone: 'neutral', when: 'withdrawn' }
    case 'published': {
      const published = `published ${timeAgo(item.published_at ?? item.created_at, now)}`
      if (expired && item.expires_at) {
        return { ...base, tag: 'Expired', tone: 'neutral', when: `expired ${shortDate(item.expires_at)}` }
      }
      const until = item.expires_at ? ` · until ${shortDate(item.expires_at)}` : ''
      let edit = ''
      if (item.edit?.status === 'pending') edit = ` · your edit is with ${approverPhrase(item.edit.approver)}`
      return {
        ...base,
        tag: 'Live',
        tone: 'live',
        when: published + until + edit,
        needsAuthor: item.edit?.status === 'rejected',
      }
    }
  }
}

// Where an announcement row in My posts leads.
export function hrefFor(item: Authored, now = new Date()) {
  return standing(item, now).opensInComposer ? `/mine/${item.id}/edit` : `/mine/${item.id}`
}

// approverPhrase names an approver inside a sentence: "the CS HOD",
// "the principal or an admin".
export function approverPhrase(approver: string | null | undefined) {
  if (!approver) return 'the approver'
  if (approver === 'Principal or admin') return 'the principal or an admin'
  return `the ${approver}`
}

const HOUR = 3600_000

// waited says how long something has been waiting for approval. Over three
// days counts as long, and is shown in the warning colour.
export function waited(iso: string, now = new Date()) {
  const hours = Math.max(1, Math.round((now.getTime() - new Date(iso).getTime()) / HOUR))
  const days = Math.round(hours / 24)
  const text = hours < 24 ? (hours === 1 ? 'waiting 1 hour' : `waiting ${hours} hours`) : days === 1 ? 'waiting 1 day' : `waiting ${days} days`
  return { text, long: hours > 72 }
}

// hasExpired is true once an item's expiry date has passed; a queue item
// like that can only be sent back.
export function hasExpired(item: { expires_at: string | null }, now = new Date()) {
  return item.expires_at !== null && new Date(item.expires_at) <= now
}
