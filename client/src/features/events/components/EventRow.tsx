import { Clock, MapPin } from 'lucide-react'
import { Link } from 'react-router-dom'
import { happeningNow, isFull, seatsLine, timeRange } from '../format'
import { typeLabel, type CampusEvent } from '../types'
import { DateTile } from './DateTile'
import { Tag } from './Tags'

// answerTag says where the reader stands: going, interested, cancelled or full.
function AnswerTag({ event }: { event: CampusEvent }) {
  const going = event.rsvp?.counts.going ?? 0
  if (event.status === 'cancelled') return <Tag tone="cancelled">Cancelled</Tag>
  if (happeningNow(event)) return <Tag tone="going">Happening now</Tag>
  const over = new Date(event.ends_at) <= new Date()
  if (over) return event.rsvp?.my_status === 'going' ? <Tag>You said going</Tag> : null
  if (event.rsvp?.my_status === 'going') return <Tag tone="going">You're going</Tag>
  if (event.rsvp?.my_status === 'interested') return <Tag>Interested</Tag>
  if (isFull(event.capacity, going)) return <Tag>Full</Tag>
  return null
}

function countText(event: CampusEvent) {
  if (event.status === 'cancelled') return event.rsvp?.my_status === 'going' ? 'you said going' : ''
  if (happeningNow(event) || new Date(event.ends_at) <= new Date()) return `${event.rsvp?.counts.going ?? 0} going`
  return seatsLine(event.capacity, event.rsvp?.counts.going ?? 0)
}

// EventRow is one Event in a list. The whole row opens it; the date tile
// and times are what people scan for.
export function EventRow({ event }: { event: CampusEvent }) {
  const time = timeRange(event.starts_at, event.ends_at)
  return (
    <li>
      <Link
        to={`/events/${event.id}`}
        className="flex gap-3 px-3.5 py-3 text-ink hover:bg-paper hover:text-ink lg:items-center lg:gap-[18px] lg:rounded-[10px] lg:py-3 lg:pr-4 lg:pl-3"
      >
        <DateTile iso={event.starts_at} />
        <span className="flex min-w-0 flex-1 flex-col gap-1 lg:gap-[5px]">
          <span className="order-3 flex items-center gap-1.5 lg:order-none">
            <Tag>{typeLabel(event.event_type)}</Tag>
            {event.department && <Tag tone="department">{event.department.code}</Tag>}
            <span className="lg:hidden">
              <AnswerTag event={event} />
            </span>
          </span>
          <span className="text-base leading-[1.3] font-semibold lg:text-[17px]">{event.title}</span>
          <span className="text-[13px] text-ink-3 lg:hidden">
            {time} · {event.location}
          </span>
          <span className="hidden gap-4 text-[13px] text-ink-2 lg:flex">
            <span className="inline-flex items-center gap-1.5">
              <Clock aria-hidden="true" className="size-4" />
              {time}
            </span>
            <span className="inline-flex min-w-0 items-center gap-1.5">
              <MapPin aria-hidden="true" className="size-4 shrink-0" />
              <span className="truncate">{event.location}</span>
            </span>
          </span>
        </span>
        <span className="hidden shrink-0 items-center gap-2 lg:flex">
          <AnswerTag event={event} />
          <span className="font-mono text-xs whitespace-nowrap text-ink-3">{countText(event)}</span>
        </span>
      </Link>
    </li>
  )
}
