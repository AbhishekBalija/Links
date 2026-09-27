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
import { useDashboard, type Dashboard } from '../api'

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
  const { user, notices, approvals, my_announcements: mine } = data
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
        {approvals && <ReviewPanel approvals={approvals} />}
        {mine && <MinePanel mine={mine} />}
        <LatestNotices notices={notices.items} />
      </div>
    </>
  )
}

// ReviewPanel comes first for approvers: reviewing is their main job here.
// It shows the three oldest items; each opens straight in the queue.
function ReviewPanel({ approvals }: { approvals: NonNullable<Dashboard['approvals']> }) {
  const count = approvals.pending_count
  const queue = useQueue()
  const oldest = (queue.data?.pages[0]?.data ?? []).slice(0, 3)
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
            const wait = waited(item.submitted_at)
            return (
              <li key={item.id}>
                <Link
                  to={`/approvals/${item.id}`}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 rounded-lg px-3 py-3 text-ink hover:bg-paper hover:text-ink"
                >
                  <span className="flex flex-col gap-0.5">
                    <span className="text-[15px] font-semibold lg:text-base">{item.title}</span>
                    <span className="text-[13px] text-ink-3">
                      {item.publisher_name} · {audienceLabel(item.audience)}
                      {item.kind === 'edit' && ' · edit to a live notice'}
                    </span>
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
