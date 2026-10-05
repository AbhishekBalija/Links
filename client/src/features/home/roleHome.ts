import type { Dashboard } from './api'

// HomeKind is which Home a user gets. The HOD, the principal, admins and
// faculty each get one built around their normal job; everyone else gets the
// reader's Home. Someone with several of these roles gets the most senior one.
export type HomeKind = 'admin' | 'principal' | 'hod' | 'faculty' | 'everyone'

export function homeKind(roles: string[]): HomeKind {
  if (roles.includes('admin')) return 'admin'
  if (roles.includes('principal')) return 'principal'
  if (roles.includes('hod')) return 'hod'
  // Placement staff keep the Home built for drives. Student coordinators
  // keep the student Home: they are students first.
  if (roles.includes('faculty') && !roles.includes('placement_officer')) return 'faculty'
  return 'everyone'
}

const words = ['No', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten']

// spell writes small numbers out, "Two", as the boards do.
export function spell(n: number): string {
  return n < words.length ? words[n] : String(n)
}

// count reads "Two students are" or "One student is", spelling out small
// numbers as the boards do.
function count(n: number, one: string, many: string, verbs: [string, string] = ['is', 'are']): string {
  return `${spell(n)} ${n === 1 ? one : many} ${n === 1 ? verbs[0] : verbs[1]}`
}

function lower(text: string): string {
  return text.charAt(0).toLowerCase() + text.slice(1)
}

// summaryLine is the sentence under the greeting: what waits for this
// person today, or that nothing does.
export function summaryLine(kind: HomeKind, data: Dashboard): string {
  const requests = data.access_requests?.pending_count ?? 0
  const announcements = data.approvals?.pending_count ?? 0
  const events = data.approvals?.events_pending_count ?? 0
  const parts: string[] = []

  if (kind === 'hod') {
    if (requests > 0) parts.push(`${count(requests, 'student', 'students')} waiting to get in`)
    const posts = announcements + events
    if (posts > 0) parts.push(`${count(posts, 'post', 'posts', ['needs', 'need'])} your approval`)
    if (parts.length === 0) return 'Nothing is waiting for you.'
    return parts.length === 2 ? `${parts[0]}, and ${lower(parts[1])}.` : `${parts[0]}.`
  }

  if (kind === 'principal') {
    const sentences: string[] = []
    if (events > 0 && announcements > 0) {
      const what = (n: number, one: string, many: string) => `${n < words.length ? words[n] : n} ${n === 1 ? one : many}`
      sentences.push(`${what(events, 'event', 'events')} and ${lower(what(announcements, 'announcement', 'announcements'))} are waiting for your approval.`)
    } else if (events > 0) {
      sentences.push(`${count(events, 'event', 'events')} waiting for your approval.`)
    } else if (announcements > 0) {
      sentences.push(`${count(announcements, 'announcement', 'announcements')} waiting for your approval.`)
    }
    const noHOD = (data.college?.departments ?? []).filter((d) => !d.hod)
    if (noHOD.length === 1) sentences.push(`${noHOD[0].name} has no HOD.`)
    else if (noHOD.length > 1) sentences.push(`${count(noHOD.length, 'department', 'departments', ['has', 'have'])} no HOD.`)
    return sentences.length > 0 ? sentences.join(' ') : 'Nothing is waiting for you.'
  }

  if (kind === 'admin') {
    if (requests > 0) return `${count(requests, 'person', 'people')} waiting to get in.`
    return 'Nobody is waiting to get in.'
  }

  return ''
}

// yourPosts is the person's own posts that need them: drafts not yet sent
// and posts sent back with a note. Posts waiting on someone else don't
// count, so most days it is null and the row stays hidden.
export function yourPosts(mine: Dashboard['my_announcements']): { count: number; detail: string } | null {
  if (!mine) return null
  const count = mine.draft + mine.rejected
  if (count === 0) return null
  const parts: string[] = []
  if (mine.draft > 0) parts.push(`${mine.draft} ${mine.draft === 1 ? 'draft' : 'drafts'}`)
  if (mine.rejected > 0) parts.push(`${mine.rejected} sent back with a note`)
  return { count, detail: parts.join(', ') }
}
