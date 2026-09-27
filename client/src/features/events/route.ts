import type { EventStatus, EventType } from './types'

type RouteInput = {
  roles: string[]
  eventType: EventType
  department: { id: string; code: string } | null
  // Whether the event's Department has an HOD to review it.
  hasHOD: boolean
  // Set when a proposal sent back for changes goes in again.
  resubmitting?: Extract<EventStatus, 'hod_changes_requested' | 'final_changes_requested'>
}

export type Route = { sentence: string; button: string }

// routeFor says what submitting does, following the server's rules
// (ADR 0023): the principal and admins publish straight away; an HOD, or the
// placement office with a training event, skips the HOD stage; everyone else
// goes to the Department's HOD first, or to the principal when there is none.
export function routeFor({ roles, eventType, department, hasHOD, resubmitting }: RouteInput): Route {
  if (roles.includes('principal') || roles.includes('admin')) {
    return { sentence: 'Publishes now. Everyone invited sees it straight away.', button: 'Publish' }
  }
  const hod = department ? `the ${department.code} HOD` : 'the HOD'
  if (resubmitting === 'final_changes_requested') {
    return { sentence: 'Goes back to the principal for final approval.', button: 'Resubmit' }
  }
  if (resubmitting === 'hod_changes_requested') {
    return { sentence: `Goes back to ${hasHOD ? hod : 'the principal'}, then the principal.`, button: 'Resubmit' }
  }
  const button = 'Submit for approval'
  if (roles.includes('hod')) {
    return { sentence: 'Goes to the principal for final approval. Your approval as HOD is already counted.', button }
  }
  if (roles.includes('placement_officer') && eventType === 'training') {
    return { sentence: 'Goes to the principal for final approval.', button }
  }
  if (!hasHOD) {
    return { sentence: `No ${department ? `${department.code} ` : ''}HOD is listed, so the principal does both reviews.`, button }
  }
  return { sentence: `Goes to ${hod}, then the principal. It's published once both approve.`, button }
}
