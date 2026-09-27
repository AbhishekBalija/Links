export type EventType = 'talk' | 'workshop' | 'competition' | 'cultural' | 'sports' | 'training' | 'other'

export const eventTypes: { value: EventType; label: string }[] = [
  { value: 'talk', label: 'Talk' },
  { value: 'workshop', label: 'Workshop' },
  { value: 'competition', label: 'Competition' },
  { value: 'cultural', label: 'Cultural' },
  { value: 'sports', label: 'Sports' },
  { value: 'training', label: 'Training' },
  { value: 'other', label: 'Other' },
]

export function typeLabel(type: EventType) {
  return eventTypes.find((t) => t.value === type)?.label ?? type
}

export function isEventType(value: string | null): value is EventType {
  return eventTypes.some((t) => t.value === value)
}

export type Answer = 'going' | 'interested' | 'not_going'

export type AnswerCounts = { going: number; interested: number; not_going: number }

export type AnswerSummary = { counts: AnswerCounts; my_status: Answer | null }

export type EventAudienceRule = {
  department_id?: string
  department_code?: string
  batch_year?: number
  role?: string
}

export type CampusEvent = {
  id: string
  title: string
  description: string
  event_type: EventType
  status: string
  proposer_id: string
  proposer_name: string
  department: { id: string; code: string } | null
  faculty_mentor: { user_id: string; full_name: string } | null
  location: string
  starts_at: string
  ends_at: string
  capacity: number | null
  audience: EventAudienceRule[] | null
  cancelled_at?: string
  cancel_reason?: string
  // Set on feed items only.
  rsvp?: AnswerSummary
}

export type Show = 'upcoming' | 'going' | 'past'

export type FeedMeta = { next_cursor?: string }
