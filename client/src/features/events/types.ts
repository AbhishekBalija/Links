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

export const answerLabels: Record<Answer, string> = { going: 'Going', interested: 'Interested', not_going: "Can't go" }

export type AnswerCounts = { going: number; interested: number; not_going: number }

// One person's answer, as an event's organisers see it.
export type AnswerPerson = { user_id: string; full_name: string; username: string; status: Answer; responded_at: string }

// Organisers also get the people who answered, earliest first.
export type AnswerSummary = { counts: AnswerCounts; my_status: Answer | null; people?: AnswerPerson[] }

export type EventAudienceRule = {
  department_id?: string
  department_code?: string
  batch_year?: number
  role?: string
}

export type EventStatus =
  | 'draft'
  | 'submitted'
  | 'hod_changes_requested'
  | 'hod_rejected'
  | 'hod_approved'
  | 'final_changes_requested'
  | 'final_rejected'
  | 'published'
  | 'cancelled'

// One reviewer's decision, at the HOD stage or at final approval.
export type Review = {
  stage: 'hod' | 'final'
  decision: 'approve' | 'request_changes' | 'reject'
  note?: string
  reviewer_name: string
  decided_at: string
}

export type CampusEvent = {
  id: string
  title: string
  description: string
  event_type: EventType
  status: EventStatus
  proposer_id: string
  proposer_name: string
  department: { id: string; code: string } | null
  faculty_mentor: { user_id: string; full_name: string } | null
  location: string
  starts_at: string
  ends_at: string
  capacity: number | null
  audience: EventAudienceRule[] | null
  submitted_at?: string
  published_at?: string
  cancelled_at?: string
  cancel_reason?: string
  created_at: string
  updated_at: string
  // Every decision so far, oldest first. Empty for readers.
  reviews?: Review[]
  // Set on feed items only.
  rsvp?: AnswerSummary
}

export type Show = 'upcoming' | 'going' | 'past'

export type FeedMeta = { next_cursor?: string }

// The proposer's own list, one status group at a time (GET /events/mine).
export type MineEventFilter = 'draft' | 'waiting' | 'attention' | 'live' | 'ended'

// What the proposal form sends. Dates are ISO strings; null capacity means
// no limit.
export type EventInput = {
  title: string
  description: string
  event_type: EventType
  department_id: string | null
  location: string
  starts_at: string
  ends_at: string
  capacity: number | null
  audience: { department_id?: string; batch_year?: number; role?: string }[]
}

// An event waiting in the reviewer's queue, at the HOD stage or at final
// approval (GET /events/reviews).
export type EventQueueItem = CampusEvent & { stage: 'hod' | 'final' }
