import { Link } from 'react-router-dom'
import { Avatar } from '../../../app/shell/Avatar'
import { Skeleton } from '../../../shared/ui/states'
import { highlight, roleLine } from '../labels'
import type { Entry } from '../types'

type Props = {
  person: Entry
  // The signed-in member's username, whose row opens their own profile.
  me?: string | null
  // The search, whose match is marked in the name.
  q?: string
  // Where the profile's back link returns to, so filters survive the trip.
  back?: string
}

// PersonRow is one member in a list: name and who they are, plus their
// headline on desktop. The whole row opens their profile.
export function PersonRow({ person, me, q, back }: Props) {
  const own = me !== undefined && me !== null && person.username === me
  const to = own ? '/profile' : `/people/${encodeURIComponent(person.username)}`
  const match = q ? highlight(person.full_name, q) : null
  // A search can also find someone by username, headline or a near spelling;
  // the tag says why they are listed when the name doesn't contain it.
  const similar = Boolean(q) && !match && !`${person.username} ${person.headline ?? ''}`.toLowerCase().includes((q ?? '').toLowerCase())

  return (
    <li>
      <Link
        to={to}
        state={back ? { back } : undefined}
        className="flex min-h-[60px] items-center gap-3 px-3.5 py-2 text-ink hover:bg-paper hover:text-ink lg:grid lg:grid-cols-[40px_minmax(0,1.1fr)_minmax(0,1fr)] lg:gap-4 lg:rounded-lg"
      >
        <Avatar name={person.full_name} className="size-10 text-sm" />
        <span className="flex min-w-0 flex-col gap-px lg:gap-0.5">
          <span className="truncate text-base font-semibold lg:text-[15px]">
            {match ? (
              <>
                {match.before}
                <mark className="rounded-sm bg-well font-bold text-inherit">{match.match}</mark>
                {match.after}
              </>
            ) : (
              person.full_name
            )}
            {similar && <span className="ml-1.5 rounded bg-well px-[7px] py-0.5 align-[1px] text-[11px] font-semibold text-ink-2">Similar spelling</span>}
            {own && <span className="ml-1.5 text-[13px] font-normal text-ink-3">(you)</span>}
          </span>
          <span className="truncate text-[13px] text-ink-3">
            <span className="lg:hidden">{roleLine(person, { short: true })}</span>
            <span className="hidden lg:inline">{roleLine(person)}</span>
          </span>
        </span>
        <span className="hidden truncate text-[13px] text-ink-2 lg:block">{person.headline}</span>
      </Link>
    </li>
  )
}

export function PersonRowSkeleton() {
  return (
    <li aria-hidden="true" className="flex min-h-[60px] items-center gap-3 px-3.5 py-2">
      <Skeleton className="size-10 rounded-full" />
      <span className="flex flex-1 flex-col gap-1.5">
        <Skeleton className="h-3.5 w-[55%] max-w-60" />
        <Skeleton className="h-3 w-[35%] max-w-40" />
      </span>
    </li>
  )
}
