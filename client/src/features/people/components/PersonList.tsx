import { byLetter } from '../labels'
import type { Entry } from '../types'
import { PersonRow, PersonRowSkeleton } from './PersonRow'

type Props = {
  people: Entry[]
  me?: string | null
  // With a search the list is ranked, so it has no letter headings.
  q?: string
  back?: string
}

const panel = 'flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:overflow-visible lg:rounded-none lg:border-0 lg:bg-transparent'
const rows = '[&>li+li]:shadow-[0_-1px_0_#efe9de] lg:[&>li+li]:shadow-none'

// PersonList shows members under letter headings, like an index. On phones
// each letter gets its own panel; on desktop one panel holds them all.
export function PersonList({ people, me, q, back }: Props) {
  if (q) {
    return (
      <div className="lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-1.5">
        <ul aria-label="Matches" className={`${panel} ${rows}`}>
          {people.map((person) => (
            <PersonRow key={person.username} person={person} me={me} q={q} back={back} />
          ))}
        </ul>
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-2 lg:gap-0.5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-1.5 lg:pb-2.5">
      {byLetter(people).map((group) => (
        <section key={group.letter} aria-labelledby={`letter-${group.letter}`} className="flex flex-col gap-2 lg:gap-0.5">
          <h2 id={`letter-${group.letter}`} className="mx-1 mt-1.5 font-serif text-[22px] font-medium text-ink-3 lg:mx-0 lg:mt-2.5 lg:px-3.5 lg:text-2xl">
            {group.letter}
          </h2>
          <ul className={`${panel} ${rows}`}>
            {group.people.map((person) => (
              <PersonRow key={person.username} person={person} me={me} back={back} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  )
}

export function PersonListSkeleton({ count = 6 }: { count?: number }) {
  return (
    <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:p-1.5">
      {Array.from({ length: count }, (_, i) => (
        <PersonRowSkeleton key={i} />
      ))}
    </ul>
  )
}
