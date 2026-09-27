import { checkProposal, toInput, type ProposalErrors, type ProposalForm } from './proposal'
import type { CampusEvent, EventInput } from './types'

// What organisers may still change once an event is published (ADR 0023).
export type LogisticsInput = Partial<Pick<EventInput, 'description' | 'location' | 'starts_at' | 'ends_at' | 'capacity'>>

const fields = ['description', 'location', 'starts_at', 'ends_at', 'capacity'] as const

// logisticsChanges is the edit to send: only the fields that changed, since
// the server refuses anything else on a published event.
export function logisticsChanges(form: ProposalForm, event: CampusEvent): LogisticsInput {
  const next = toInput(form, event.department)
  const changes: LogisticsInput = {}
  for (const field of fields) {
    const before = field === 'starts_at' || field === 'ends_at' ? new Date(event[field]).toISOString() : event[field]
    if (next[field] !== before) Object.assign(changes, { [field]: next[field] })
  }
  return changes
}

// checkLogistics finds what must be fixed before saving. A start that moves
// must still be ahead, and a seat limit can't drop below those going.
export function checkLogistics(form: ProposalForm, event: CampusEvent, now: Date, going: number): ProposalErrors {
  const errors = checkProposal(form, now, { submitting: false })
  if (!errors.starts && !errors.ends && logisticsChanges(form, event).starts_at && new Date(toInput(form, event.department).starts_at) <= now) {
    errors.starts = "That's in the past."
  }
  if (!errors.capacity && form.limitSeats && Number(form.capacity) < going) {
    errors.capacity = `${going} are going. A limit can't be lower than that.`
  }
  return errors
}
