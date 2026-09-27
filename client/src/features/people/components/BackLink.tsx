import { ChevronLeft } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'

// BackLink returns to the list the member came from, filters and all, or to
// People when they arrived from somewhere else.
export function BackLink() {
  const state = useLocation().state as { back?: string } | null
  return (
    <Link to={state?.back ?? '/people'} className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
      <ChevronLeft aria-hidden="true" className="size-5" />
      People
    </Link>
  )
}
