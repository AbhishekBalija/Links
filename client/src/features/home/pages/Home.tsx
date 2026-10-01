import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Avatar } from '../../../app/shell/Avatar'
import { LogoutButton } from '../../../app/shell/LogoutButton'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useAuthStore } from '../../auth/store'
import { NoticeList, NoticeListSkeleton } from '../../notices/components/NoticeList'
import { useRefetchAtExpiry } from '../../notices/useRefetchAtExpiry'
import { useQueue } from '../../announcements/api'
import { waited } from '../../announcements/status'
import { audienceLabel } from '../../notices/format'
import { useEventFeed, useEventReviews } from '../../events/api'
import { EventRow } from '../../events/components/EventRow'
import { useDashboard, type Dashboard } from '../api'
import { waitingForReview } from '../review'
import { mergeOldestFirst } from '../../posts/merge'
import { buttonStyles } from '../../announcements/buttons'
import { JobRow } from '../../jobs/components/JobRow'
import { daysLeft, isUrgent } from '../../jobs/format'
import { Pipeline } from '../../placement/components/Pipeline'
import { placementLine, reviewFirst, showOpenJobs, type PlacementSummary } from '../placement'

function greeting(now: Date) {
  const hour = now.getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

function firstName(fullName: string) {
  return fullName.trim().split(/\s+/)[0] ?? ''
}

export default function Home() {
  const dashboard = useDashboard()
  useRefetchAtExpiry(dashboard.data?.notices.items.map((n) => n.expires_at) ?? [], dashboard.refetch)
  const now = new Date()

  return (
    <div className="flex flex-col gap-5 lg:gap-8">
      <PhoneBar />
      {dashboard.isPending ? (
        <HomeSkeleton />
      ) : dashboard.isError ? (
        <ErrorState message="Your Home page could not be loaded." onRetry={() => dashboard.refetch()} />
      ) : (
        <HomeView data={dashboard.data} now={now} />
      )}
    </div>
  )
}

// On phones there is no sidebar, so Home carries the wordmark, the avatar
// (to the profile) and log out.
function PhoneBar() {
  const user = useAuthStore((s) => s.user)
  const name = user?.profile.full_name ?? user?.email ?? ''
  return (
    <div className="-mt-1 flex items-center justify-between lg:hidden">
      <span className="px-1 font-serif text-[26px] font-semibold tracking-[-0.4px] text-ink">Links</span>
      <span className="flex items-center gap-1">
        <LogoutButton iconOnly />
        <Link to="/profile" aria-label="Your profile" className="flex min-h-11 min-w-11 items-center justify-center rounded-full">
          <Avatar name={name} />
        </Link>
      </span>
    </div>
  )
}

function HomeView({ data, now }: { data: Dashboard; now: Date }) {
  const { user, notices, approvals, my_announcements: mine, opportunities, placement } = data
  const dateLine = now.toLocaleDateString('en-IN', { weekday: 'long', day: 'numeric', month: 'long' })

  return (
    <>
      <header className="flex flex-col gap-1.5 px-1 lg:gap-2 lg:px-0">
        <p className="font-mono text-[11px] uppercase tracking-[1.2px] text-ink-3 lg:text-xs">
          {dateLine}
          {user.department && (
            <>
              {' · '}
              <Link to={`/departments/${user.department.code}`} className="text-ink-3 hover:text-rust">
                {user.department.code}
              </Link>
            </>
          )}
        </p>
        <h1 className="font-serif text-[30px] leading-[1.1] font-medium tracking-[-0.5px] lg:text-[44px] lg:leading-[1.08] lg:tracking-[-0.8px]">
          {greeting(now)}, {firstName(user.full_name)}
        </h1>
      </header>

      {/* Phones stack review, your announcements, then notices. Desktop moves
          your announcements into a side column. */}
      <div className={cn('grid items-start gap-5 lg:gap-7', mine && 'lg:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)]')}>
        {placement && !approvals && <PlacementPanel summary={placement} now={now} />}
        {approvals && <ReviewPanel approvals={approvals} />}
        {placement && approvals && <PlacementPanel summary={placement} now={now} />}
        {mine && <MinePanel mine={mine} />}
        {opportunities && showOpenJobs(user.roles, opportunities) && <OpenJobs section={opportunities} />}
        <ComingUp />
        <LatestNotices notices={notices.items} />
      </div>
    </>
  )
}

// ReviewPanel comes first for approvers: reviewing is their main job here.
// It counts announcements and event proposals together and shows the three
// oldest of either kind; each opens straight in the queue.
function ReviewPanel({ approvals }: { approvals: NonNullable<Dashboard['approvals']> }) {
  const count = waitingForReview(approvals)
  const queue = useQueue()
  const events = useEventReviews()
  const oldest = mergeOldestFirst<Waiting>([
    {
      items: (queue.data?.pages[0]?.data ?? []).map((item) => ({
        id: item.id,
        at: item.submitted_at,
        title: item.title,
        detail: `${item.publisher_name} · ${audienceLabel(item.audience)}${item.kind === 'edit' ? ' · edit to a live notice' : ''}`,
      })),
      complete: !queue.hasNextPage,
    },
    {
      items: (events.data?.pages[0]?.data ?? []).map((item) => ({
        id: item.id,
        at: item.submitted_at ?? item.updated_at,
        title: item.title,
        detail: `${item.proposer_name} · event proposal${item.stage === 'final' ? ', final approval' : ''}`,
      })),
      complete: !events.hasNextPage,
    },
  ]).items.slice(0, 3)
  return (
    <section aria-labelledby="review-h" className="flex flex-col gap-1.5 rounded-xl border border-line bg-surface px-5 py-5 lg:col-start-1 lg:px-7 lg:py-6">
      <div className="flex items-baseline justify-between gap-3 pb-1">
        <div className="flex items-baseline gap-3">
          <h2 id="review-h" className="font-serif text-xl font-medium lg:text-2xl">
            Waiting for your review
          </h2>
          <span className="font-mono text-sm text-rust">{count}</span>
        </div>
        {count > 0 && (
          <Link to="/approvals" className="shrink-0 text-sm font-semibold">
            <span className="lg:hidden">Queue →</span>
            <span className="hidden lg:inline">Open approval queue →</span>
          </Link>
        )}
      </div>
      {count === 0 ? (
        <p className="text-[15px] text-ink-2">Nothing is waiting for you.</p>
      ) : (
        <ul className="-mx-3 flex flex-col">
          {oldest.map((item) => {
            const wait = waited(item.at)
            return (
              <li key={item.id}>
                <Link
                  to={`/approvals/${item.id}`}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 rounded-lg px-3 py-3 text-ink hover:bg-paper hover:text-ink"
                >
                  <span className="flex flex-col gap-0.5">
                    <span className="text-[15px] font-semibold lg:text-base">{item.title}</span>
                    <span className="text-[13px] text-ink-3">{item.detail}</span>
                  </span>
                  <span className={cn('font-mono text-xs whitespace-nowrap', wait.long ? 'text-warning' : 'text-ink-3')}>{wait.text}</span>
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

type Waiting = { id: string; at: string; title: string; detail: string }

// OpenJobs is the next Opportunities a student can apply to, soonest
// deadline first, so a new drive is seen without opening Jobs.
function OpenJobs({ section }: { section: NonNullable<Dashboard['opportunities']> }) {
  return (
    <section aria-labelledby="jobs-h" className="flex flex-col gap-3 lg:col-start-1">
      <div className="flex items-baseline justify-between px-1 lg:px-0">
        <h2 id="jobs-h" className="font-serif text-xl font-medium">
          Open jobs for you
        </h2>
        <Link to="/jobs" className="text-sm font-semibold">
          {section.has_more ? 'All open jobs →' : 'Jobs →'}
        </Link>
      </div>
      <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:gap-0.5 lg:p-1.5 [&>li+li]:shadow-[0_-1px_0_#efe9de] lg:[&>li+li]:shadow-none">
        {section.items.map((job) => (
          <JobRow key={job.id} job={job} />
        ))}
      </ul>
    </section>
  )
}

// PlacementPanel is the placement office's work on Home: what waits for
// review, where to start, and the open drives with their applicants.
function PlacementPanel({ summary, now }: { summary: PlacementSummary; now: Date }) {
  const first = reviewFirst(summary.drives)
  return (
    <section aria-labelledby="drives-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface px-5 py-5 lg:col-start-1 lg:px-7 lg:py-6">
      <div className="flex items-baseline justify-between gap-3">
        <div className="flex items-baseline gap-3">
          <h2 id="drives-h" className="font-serif text-xl font-medium lg:text-2xl">
            Open drives
          </h2>
          <span className="font-mono text-sm text-rust">{summary.open_count}</span>
        </div>
        <Link to="/placement" className="shrink-0 text-sm font-semibold">
          <span className="lg:hidden">Placement →</span>
          <span className="hidden lg:inline">All of Placement →</span>
        </Link>
      </div>
      <p className="text-[15px] text-ink-2">{placementLine(summary, now)}</p>
      {first && (
        <Link to={`/placement/${first.id}/applicants?status=applied`} className={cn(buttonStyles.secondary, 'self-start')}>
          Review {first.company} first · {first.applicant_counts.applied} waiting
        </Link>
      )}
      {summary.drives.length > 0 ? (
        <ul className="-mx-3 flex flex-col">
          {summary.drives.map((drive) => (
            <li key={drive.id}>
              <Link
                to={`/placement/${drive.id}`}
                className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1.5 rounded-lg px-3 py-3 text-ink hover:bg-paper hover:text-ink lg:grid-cols-[minmax(0,1fr)_120px_220px]"
              >
                <span className="flex flex-col gap-0.5">
                  <span className="text-[15px] font-semibold lg:text-base">{drive.title}</span>
                  <span className="text-[13px] text-ink-3">{drive.company}</span>
                </span>
                <span className={cn('font-mono text-xs whitespace-nowrap', isUrgent(drive.apply_by, now) ? 'text-warning' : 'text-ink-3')}>
                  {daysLeft(drive.apply_by, now)}
                </span>
                <Pipeline counts={drive.applicant_counts} className="col-span-2 lg:col-span-1" />
              </Link>
            </li>
          ))}
        </ul>
      ) : (
        <Link to="/placement/new" className={cn(buttonStyles.primary, 'self-start hover:text-paper')}>
          New opportunity
        </Link>
      )}
    </section>
  )
}

// ComingUp shows the next two Events for the reader. Home stays quiet when
// nothing is coming up, so the block only appears with something in it.
function ComingUp() {
  const feed = useEventFeed('upcoming', null, 2)
  const events = feed.data?.pages[0]?.data ?? []
  if (events.length === 0) return null
  return (
    <section aria-labelledby="coming-h" className="flex flex-col gap-3 lg:col-start-1">
      <div className="flex items-baseline justify-between px-1 lg:px-0">
        <h2 id="coming-h" className="font-serif text-xl font-medium">
          Coming up
        </h2>
        <Link to="/events" className="text-sm font-semibold">
          All events →
        </Link>
      </div>
      <ul className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:gap-0.5 lg:p-1.5 [&>li+li]:shadow-[0_-1px_0_#efe9de] lg:[&>li+li]:shadow-none">
        {events.map((event) => (
          <EventRow key={event.id} event={event} />
        ))}
      </ul>
    </section>
  )
}

function LatestNotices({ notices }: { notices: Dashboard['notices']['items'] }) {
  return (
    <section aria-labelledby="latest-h" className="flex flex-col gap-3 lg:col-start-1">
      <div className="flex items-baseline justify-between px-1 lg:px-0">
        <h2 id="latest-h" className="font-serif text-xl font-medium">
          Latest notices
        </h2>
        <Link to="/notices" className="text-sm font-semibold">
          All notices →
        </Link>
      </div>
      {notices.length === 0 ? (
        <EmptyState title="No notices yet">When something is posted for you, it shows up here.</EmptyState>
      ) : (
        <NoticeList notices={notices} />
      )}
    </section>
  )
}

// MinePanel counts the user's own announcements that need them. A notice sent
// back with a note is the one to act on, so it is the only one tinted.
function MinePanel({ mine }: { mine: NonNullable<Dashboard['my_announcements']> }) {
  const counts = [
    { label: 'Draft', value: mine.draft, alert: false },
    { label: 'Waiting for approval', value: mine.pending, alert: false },
    { label: 'Sent back with a note', value: mine.rejected, alert: mine.rejected > 0 },
    { label: 'Edits waiting', value: mine.edits_waiting, alert: false },
  ]
  return (
    <section aria-labelledby="mine-h" className="flex lg:col-start-2 lg:row-span-2 lg:row-start-1 flex-col gap-3.5 rounded-xl border border-line bg-surface p-5 lg:p-6">
      <div className="flex items-baseline justify-between gap-3">
        <h2 id="mine-h" className="font-serif text-xl font-medium lg:text-[21px]">
          Your announcements
        </h2>
        <Link to="/mine" className="text-sm font-semibold">
          Open →
        </Link>
      </div>
      <dl className="grid grid-cols-2 gap-2.5">
        {counts.map((c) => (
          <div
            key={c.label}
            className={cn('flex flex-col-reverse gap-0.5 rounded-lg px-3.5 py-3', c.alert ? 'bg-danger-soft' : 'bg-paper')}
          >
            <dt className={cn('text-[13px]', c.alert ? 'font-semibold text-danger-ink' : 'text-ink-2')}>{c.label}</dt>
            <dd className={cn('font-mono text-[22px] font-medium', c.alert && 'text-danger')}>{c.value}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

function HomeSkeleton() {
  return (
    <>
      <LoadingStatus label="Loading your Home page" />
      <div aria-hidden="true" className="flex flex-col gap-2 px-1 lg:px-0">
        <Skeleton className="h-3 w-40" />
        <Skeleton className="h-9 w-72" />
      </div>
      <NoticeListSkeleton rows={4} />
    </>
  )
}
