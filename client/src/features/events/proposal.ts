import { toRules } from '../announcements/audience'
import type { RuleInput } from '../announcements/types'
import type { CampusEvent, EventInput, EventType } from './types'

// ProposalForm is the proposal form as the proposer fills it in: dates and
// times as the date and time inputs give them, in the proposer's time zone.
export type ProposalForm = {
  event_type: EventType | null
  title: string
  description: string
  location: string
  startDate: string
  startTime: string
  endDate: string
  endTime: string
  limitSeats: boolean
  capacity: string
  audience: RuleInput[]
}

export type ProposalErrors = Partial<Record<'event_type' | 'title' | 'starts' | 'ends' | 'location' | 'capacity' | 'description' | 'audience', string>>

export function emptyProposal(audience: RuleInput[], eventType: EventType | null = null): ProposalForm {
  return {
    event_type: eventType,
    title: '',
    description: '',
    location: '',
    startDate: '',
    startTime: '',
    endDate: '',
    endTime: '',
    limitSeats: false,
    capacity: '',
    audience,
  }
}

const pad = (n: number) => String(n).padStart(2, '0')
const dateInput = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
const timeInput = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}`

// moment joins a date and a time input into a local Date, or null while
// either is missing.
function moment(date: string, time: string): Date | null {
  if (!date || !time) return null
  const [y, m, d] = date.split('-').map(Number)
  const [h, min] = time.split(':').map(Number)
  return new Date(y, m - 1, d, h, min)
}

// An empty end date means the same day as the start.
const starts = (form: ProposalForm) => moment(form.startDate, form.startTime)
const ends = (form: ProposalForm) => moment(form.endDate || form.startDate, form.endTime)

export function fromEvent(event: CampusEvent): ProposalForm {
  const start = new Date(event.starts_at)
  const end = new Date(event.ends_at)
  return {
    event_type: event.event_type,
    title: event.title,
    description: event.description,
    location: event.location,
    startDate: dateInput(start),
    startTime: timeInput(start),
    endDate: dateInput(end),
    endTime: timeInput(end),
    limitSeats: event.capacity !== null,
    capacity: event.capacity === null ? '' : String(event.capacity),
    audience: toRules(event.audience ?? []),
  }
}

// toInput is what the server receives. Call it once checkProposal passes.
export function toInput(form: ProposalForm, department: { id: string; code?: string } | null): EventInput {
  const start = starts(form)
  const end = ends(form)
  if (!form.event_type || !start || !end) throw new Error('check the proposal before sending it')
  return {
    title: form.title.trim(),
    description: form.description.trim(),
    event_type: form.event_type,
    department_id: department?.id ?? null,
    location: form.location.trim(),
    starts_at: start.toISOString(),
    ends_at: end.toISOString(),
    capacity: form.limitSeats ? Number(form.capacity) : null,
    audience: form.audience,
  }
}

// checkProposal finds what must be fixed before saving. A draft may have
// past dates; submitting needs a start still ahead, as the server does.
export function checkProposal(form: ProposalForm, now: Date, { submitting }: { submitting: boolean }): ProposalErrors {
  const errors: ProposalErrors = {}
  if (!form.event_type) errors.event_type = 'Choose what kind of event it is.'
  const title = form.title.trim()
  if (!title) errors.title = 'Give the event a title.'
  else if (title.length < 3) errors.title = 'Use at least 3 characters.'

  const start = starts(form)
  const end = ends(form)
  if (!start) errors.starts = 'Choose when it starts.'
  else if (submitting && start <= now) errors.starts = "That's in the past."
  if (!end) errors.ends = 'Choose when it ends.'
  else if (start && end <= start) errors.ends = 'It has to end after it starts.'

  if (!form.location.trim()) errors.location = 'Say where it happens.'
  if (form.limitSeats && !/^[1-9]\d*$/.test(form.capacity.trim())) errors.capacity = 'Use a whole number from 1 up.'
  return errors
}

// durationLabel shows how long the event runs, beside the end time.
export function durationLabel(form: ProposalForm): string {
  const start = starts(form)
  const end = ends(form)
  if (!start || !end || end <= start) return ''
  const minutes = Math.round((end.getTime() - start.getTime()) / 60000)
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  if (start.toDateString() !== end.toDateString()) {
    const days = Math.round((new Date(end.toDateString()).getTime() - new Date(start.toDateString()).getTime()) / 86_400_000) + 1
    const span = days === 2 ? 'two days' : `${days} days`
    return `${hours} hours${rest ? ` ${rest} min` : ''}, over ${span}`
  }
  if (hours === 0) return `${rest} min`
  return rest ? `${hours} h ${rest} min` : `${hours} h`
}
