import { approverPhrase } from '../announcements/status'
import { spell } from './roleHome'

// MyWork is an author's own announcements and events that need them (sent
// back with a note) or wait on a reviewer. The server caps each list and
// says when there are more.
export type MyWork = {
  sent_back: {
    kind: 'announcement' | 'event'
    id: string
    title: string
    note: string | null
    sent_back_by: string
    sent_back_at: string
    is_edit: boolean
  }[]
  sent_back_has_more: boolean
  waiting: {
    kind: 'announcement' | 'event'
    id: string
    title: string
    waiting_on: string
    since: string
    is_edit: boolean
  }[]
  waiting_has_more: boolean
}

// How many sent back items Home shows with their notes before "more".
const shownOnHome = 3

// authorLine is the sentence under a faculty member's greeting.
export function authorLine(work: MyWork | undefined): string {
  const back = work?.sent_back.length ?? 0
  const waiting = work?.waiting.length ?? 0
  const sentences: string[] = []
  if (back > 0) sentences.push(`${spell(back)} ${back === 1 ? 'post was' : 'posts were'} sent back with a note.`)
  if (waiting > 0) {
    // After a sentence about posts, the second needn't say "post" again.
    const what = back > 0 ? '' : waiting === 1 ? ' post' : ' posts'
    sentences.push(`${spell(waiting)}${what} ${waiting === 1 ? 'is' : 'are'} waiting on others.`)
  }
  return sentences.length > 0 ? sentences.join(' ') : 'Nothing of yours is waiting on anyone.'
}

// sentBackOnHome is the first few sent back items and how many more there
// are; orMore is set when the server's list was cut short too.
export function sentBackOnHome(work: MyWork) {
  return {
    shown: work.sent_back.slice(0, shownOnHome),
    more: Math.max(0, work.sent_back.length - shownOnHome),
    orMore: work.sent_back_has_more,
  }
}

export function workHref(item: Pick<MyWork['waiting'][number], 'kind' | 'id'>): string {
  return item.kind === 'event' ? `/mine/events/${item.id}` : `/mine/${item.id}`
}

function kindLabel(kind: 'announcement' | 'event', isEdit: boolean): string {
  if (kind === 'event') return 'Event'
  return isEdit ? 'Edit to a live announcement' : 'Announcement'
}

// workDetail reads "Event · With the CS HOD".
export function workDetail(item: Pick<MyWork['waiting'][number], 'kind' | 'waiting_on' | 'is_edit'>): string {
  return `${kindLabel(item.kind, item.is_edit)} · With ${approverPhrase(item.waiting_on)}`
}

export function sentBackKind(item: MyWork['sent_back'][number]): string {
  return kindLabel(item.kind, item.is_edit).toLowerCase()
}
