import { Skeleton } from '../../../shared/ui/states'
import { groupLabel } from '../format'
import type { CampusEvent } from '../types'
import { EventRow } from './EventRow'

type Group = { label: string; events: CampusEvent[] }

function groups(events: CampusEvent[], past: boolean, now: Date): Group[] {
  const out: Group[] = []
  for (const event of events) {
    const label = groupLabel(event.starts_at, now, past)
    const last = out[out.length - 1]
    if (last && last.label === label) last.events.push(event)
    else out.push({ label, events: [event] })
  }
  return out
}

// EventList shows Events under week headings, "This week" first, each in
// its own soft panel.
export function EventList({ events, past = false }: { events: CampusEvent[]; past?: boolean }) {
  const now = new Date()
  return (
    <div className="flex flex-col gap-4 lg:gap-[18px]">
      {groups(events, past, now).map((group) => (
        <section key={group.label} aria-label={group.label} className="flex flex-col gap-1.5">
          <h2 className="mx-1 font-serif text-xl font-medium text-ink-3 lg:text-[22px]">{group.label}</h2>
          <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:gap-0.5 lg:p-1.5 [&>li+li]:shadow-[0_-1px_0_#efe9de] lg:[&>li+li]:shadow-none">
            {group.events.map((event) => (
              <EventRow key={event.id} event={event} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  )
}

export function EventListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <ul aria-hidden="true" className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface">
      {Array.from({ length: rows }, (_, i) => (
        <li key={i} className="flex gap-3 px-3.5 py-3">
          <Skeleton className="h-16 w-[52px] rounded-[10px]" />
          <span className="flex flex-1 flex-col gap-2 pt-1">
            <Skeleton className="h-4 w-4/5" />
            <Skeleton className="h-3 w-1/2" />
            <Skeleton className="h-3 w-1/4" />
          </span>
        </li>
      ))}
    </ul>
  )
}
