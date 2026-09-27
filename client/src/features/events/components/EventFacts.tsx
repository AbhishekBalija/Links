import type { ReactNode } from 'react'
import { audienceLabel } from '../../notices/format'
import { whenLine } from '../format'
import type { CampusEvent } from '../types'

// EventFacts is what the people running an event check first: when, where,
// who it's for and how many seats, then its description.
export function EventFacts({ event, children }: { event: CampusEvent; children?: ReactNode }) {
  const when = whenLine(event.starts_at, event.ends_at)
  const audience = audienceLabel(
    (event.audience ?? []).map((rule) => ({
      department_id: rule.department_id ?? null,
      department_code: rule.department_code ?? null,
      batch_year: rule.batch_year ?? null,
      role: rule.role ?? null,
    })),
  )
  return (
    <>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 rounded-xl border border-line bg-surface px-4 py-3.5 text-sm lg:gap-x-[18px] lg:gap-y-2.5 lg:border-0 lg:bg-transparent lg:p-0 lg:text-[15px]">
        <Fact label="When">
          {when.date && <b className="font-semibold">{when.date} </b>}
          <span className="font-mono text-[13px] lg:text-sm">{when.time}</span>
        </Fact>
        <Fact label="Where">{event.location}</Fact>
        <Fact label="For">{audience}</Fact>
        {children}
        <Fact label="Seats">{event.capacity === null ? 'No limit' : `${event.capacity} can say Going`}</Fact>
      </dl>
      {event.description && (
        <div className="flex max-w-[640px] flex-col gap-3 font-serif text-[17px] leading-[1.55] text-prose lg:text-lg">
          {event.description.split(/\n\s*\n/).map((paragraph, i) => (
            <p key={i} className="whitespace-pre-line">
              {paragraph}
            </p>
          ))}
        </div>
      )}
    </>
  )
}

export function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-ink-3">{label}</dt>
      <dd>{children}</dd>
    </>
  )
}
