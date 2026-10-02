import { useState, type ReactNode } from 'react'
import { ChevronLeft } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { EmptyState, ErrorState } from '../../../shared/ui/states'
import { useIsDesktop } from '../../../shared/ui/useIsDesktop'
import { QueueSkeleton } from '../../announcements/components/queue/QueueSkeleton'
import type { Banner } from '../../announcements/components/queue/Review'
import { useAccessRequests } from '../api'
import { RequestRow } from './RequestRow'
import { RequestReview } from './RequestReview'

// AccessRequestsView is the list of waiting Access requests beside the one
// selected, oldest first. On phones each request opens on its own page.
// basePath is where it lives: /approvals/access for an HOD, /admin/requests
// for an admin.
export function AccessRequestsView({ header, basePath, backLabel, empty }: {
  header: ReactNode
  basePath: string
  backLabel: string
  empty: ReactNode
}) {
  const { id } = useParams()
  const isDesktop = useIsDesktop()
  const navigate = useNavigate()
  const requests = useAccessRequests()
  const [banner, setBanner] = useState<Banner | null>(null)
  const [focusName, setFocusName] = useState(false)

  if (requests.isPending) return <QueueSkeleton header={header} />
  if (requests.isError) {
    return (
      <div className="flex flex-col gap-5">
        {header}
        <ErrorState message="Access requests could not be loaded." onRetry={() => requests.refetch()} />
      </div>
    )
  }

  const items = requests.data
  const selected = items.find((item) => item.id === id) ?? (isDesktop ? items[0] : undefined)

  // After a decision, move on: the next request on desktop, the list on phones.
  function moveOn(decidedId: string, next: Banner) {
    setBanner(next)
    setFocusName(true)
    const index = items.findIndex((item) => item.id === decidedId)
    const after = items[index + 1] ?? items.filter((item) => item.id !== decidedId)[0]
    if (isDesktop && after) navigate(`${basePath}/${after.id}`, { replace: true })
    else navigate(basePath, { replace: true })
  }

  const bannerView = banner && (
    <div
      role="status"
      className={cn('rounded-[10px] px-4 py-3 text-sm', banner.tone === 'done' ? 'bg-success-soft text-success-ink' : 'border border-line bg-surface text-ink')}
    >
      {banner.text}
    </div>
  )

  if (items.length === 0) {
    return (
      <div className="flex flex-col gap-5">
        {header}
        {bannerView}
        <EmptyState title="No access requests">{empty}</EmptyState>
      </div>
    )
  }

  if (!isDesktop && selected) {
    return (
      <div className="group/page flex flex-col gap-3 pb-44">
        <Link to={basePath} className="-ml-2 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-2 text-sm font-semibold">
          <ChevronLeft aria-hidden="true" className="size-5" />
          {backLabel}
        </Link>
        <RequestReview key={selected.id} request={selected} focusName={focusName} onDecided={(request, next) => moveOn(request.id, next)} />
      </div>
    )
  }

  return (
    <div className="group/page flex flex-col gap-5 pb-4 lg:pb-0">
      {header}
      {bannerView}
      <div className="grid items-start gap-5 lg:grid-cols-[360px_minmax(0,1fr)]">
        <nav aria-label="Access requests" className="flex flex-col gap-2.5 lg:sticky lg:top-6 lg:gap-0.5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-1.5">
          {items.map((request) => (
            <RequestRow key={request.id} request={request} selected={isDesktop && request.id === selected?.id} to={`${basePath}/${request.id}`} />
          ))}
        </nav>
        {isDesktop && selected && (
          <RequestReview key={selected.id} request={selected} focusName={focusName} onDecided={(request, next) => moveOn(request.id, next)} />
        )}
      </div>
    </div>
  )
}
