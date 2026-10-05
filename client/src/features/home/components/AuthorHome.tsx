import { Check, Plus } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { buttonStyles } from '../../announcements/buttons'
import { waited } from '../../announcements/status'
import { timeAgo } from '../../notices/format'
import { useHasPosted } from '../../posts/api'
import { NewMenu } from '../../posts/components/NewMenu'
import type { Dashboard } from '../api'
import { authorLine, sentBackKind, sentBackOnHome, workDetail, workHref, type MyWork } from '../author'

type Props = {
  data: Dashboard
  greeting: ReactNode
  comingUp: ReactNode
  notices: ReactNode
  now: Date
}

const panel = 'flex flex-col rounded-xl border border-line bg-surface px-[18px] py-4 lg:px-7 lg:py-6'

// FacultyHome is built around what faculty normally do: post and propose,
// then fix what comes back. What needs them comes first, then what waits on
// a reviewer; their Department's events and the latest notices sit beside it.
export function FacultyHome({ data, greeting, comingUp, notices, now }: Props) {
  const hasPosted = useHasPosted()
  const firstTime = hasPosted.data === false
  const drafts = data.my_announcements?.draft ?? 0

  return (
    <>
      <header className="flex flex-col gap-3 px-1 lg:flex-row lg:items-end lg:justify-between lg:px-0">
        <div className="flex flex-col gap-1.5 lg:gap-2">
          {greeting}
          <p className="text-[15px] text-ink-2 lg:text-base">{firstTime ? 'Here is what you can do on LINKS.' : authorLine(data.my_work)}</p>
        </div>
        {/* The phone's New sits in the top bar. */}
        {!firstTime && (
          <div className="hidden lg:block">
            <NewMenu />
          </div>
        )}
      </header>
      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] lg:gap-7">
        <div className="flex flex-col gap-5 lg:gap-6">
          {firstTime ? (
            <FirstTime />
          ) : (
            <>
              <NeedsYou work={data.my_work} drafts={drafts} quietWhenEmpty={false} />
              <WaitingOnOthers work={data.my_work} now={now} />
            </>
          )}
        </div>
        <div className="flex flex-col gap-5 lg:gap-6">
          {comingUp}
          {notices}
        </div>
      </div>
    </>
  )
}

// AuthorSections is the same work for a student coordinator, on top of
// their student Home. It only shows when something is there.
export function AuthorSections({ data, now }: { data: Dashboard; now: Date }) {
  const drafts = data.my_announcements?.draft ?? 0
  return (
    <>
      <NeedsYou work={data.my_work} drafts={drafts} quietWhenEmpty />
      <WaitingOnOthers work={data.my_work} now={now} />
    </>
  )
}

function NeedsYou({ work, drafts, quietWhenEmpty }: { work?: MyWork; drafts: number; quietWhenEmpty: boolean }) {
  const back = work ? sentBackOnHome(work) : { shown: [], more: 0, orMore: false }
  if (quietWhenEmpty && back.shown.length === 0 && drafts === 0) return null
  return (
    <section aria-labelledby="needs-h" className={cn(panel, 'gap-3.5')}>
      <h2 id="needs-h" className="font-serif text-xl font-medium lg:text-[22px]">
        Needs you
      </h2>
      {back.shown.length === 0 ? (
        <p className="flex items-center gap-2 text-[15px] text-ink-2">
          <Check aria-hidden="true" className="size-4 shrink-0 text-success" strokeWidth={2.2} />
          Nothing needs you right now.
        </p>
      ) : (
        <ul className="flex flex-col gap-5">
          {back.shown.map((item) => (
            <li key={`${item.kind}-${item.id}`}>
              <SentBackItem item={item} />
            </li>
          ))}
        </ul>
      )}
      {back.more > 0 && (
        <Link to="/mine?status=needs" className="text-sm font-semibold">
          {back.more}
          {back.orMore ? ' or more' : ''} more sent back in My posts →
        </Link>
      )}
      {drafts > 0 && (
        <Link
          to="/mine?status=draft"
          className="flex min-h-11 items-center justify-between gap-3 border-t border-line pt-1 text-sm text-ink-2 hover:text-ink"
        >
          <span>
            {drafts === 1 ? '1 draft' : `${drafts} drafts`} you haven't sent yet
          </span>
          <span className="font-semibold whitespace-nowrap text-rust">Finish {drafts === 1 ? 'it' : 'them'} →</span>
        </Link>
      )}
    </section>
  )
}

function SentBackItem({ item }: { item: MyWork['sent_back'][number] }) {
  return (
    <Link to={workHref(item)} className="flex flex-col gap-2.5 text-ink hover:text-ink">
      <span className="flex items-center gap-2">
        <span className="rounded bg-warning-soft px-[7px] py-0.5 text-[11px] leading-4 font-semibold whitespace-nowrap text-warning-ink">Sent back</span>
        <span className="font-mono text-xs text-ink-3">{sentBackKind(item)}</span>
      </span>
      <span className="text-base font-semibold lg:text-lg">{item.title}</span>
      {item.note && (
        <blockquote className="rounded-lg bg-paper px-3.5 py-2.5 font-serif text-base leading-normal text-ink lg:text-[17px]">
          “{item.note}”
          <span className="mt-1 block font-sans text-[13px] text-ink-3">
            {item.sent_back_by} · {timeAgo(item.sent_back_at)}
          </span>
        </blockquote>
      )}
      <span className="text-sm font-semibold text-rust">Open and fix →</span>
    </Link>
  )
}

function WaitingOnOthers({ work, now }: { work?: MyWork; now: Date }) {
  if (!work || work.waiting.length === 0) return null
  return (
    <section aria-labelledby="others-h" className={cn(panel, 'gap-1')}>
      <div className="flex items-baseline justify-between gap-3">
        <h2 id="others-h" className="font-serif text-xl font-medium lg:text-[21px]">
          Waiting on others
        </h2>
        <Link to="/mine?status=waiting" className="text-sm font-semibold">
          My posts →
        </Link>
      </div>
      <ul className="-mx-2 flex flex-col">
        {work.waiting.map((item) => (
          <li key={`${item.kind}-${item.id}`}>
            <Link to={workHref(item)} className="flex justify-between gap-3 rounded-lg px-2 py-2.5 text-ink hover:bg-paper hover:text-ink">
              <span className="flex flex-col gap-0.5">
                <span className="text-[15px] font-semibold">{item.title}</span>
                <span className="text-[13px] text-ink-3">{workDetail(item)}</span>
              </span>
              <span className="font-mono text-xs whitespace-nowrap text-ink-3">{waited(item.since, now).text.replace('waiting ', '')}</span>
            </Link>
          </li>
        ))}
      </ul>
      {work.waiting_has_more && (
        <Link to="/mine?status=waiting" className="pt-1 text-sm font-semibold">
          More in My posts →
        </Link>
      )}
    </section>
  )
}

function FirstTime() {
  const points: [string, string][] = [
    ['Write an announcement', 'for your students. Your HOD approves it before they see it.'],
    ['Propose an event.', 'Your HOD reviews it, then the principal gives final approval.'],
    ['Follow each one here,', 'and fix anything sent back with a note.'],
  ]
  return (
    <section aria-labelledby="first-h" className={cn(panel, 'gap-4')}>
      <h2 id="first-h" className="font-serif text-xl font-medium lg:text-2xl">
        Post to your classes
      </h2>
      <ul className="flex flex-col gap-2.5">
        {points.map(([bold, rest]) => (
          <li key={bold} className="flex items-start gap-3 text-[15px] leading-normal">
            <span aria-hidden="true" className="mt-[9px] size-1.5 flex-none rounded-full bg-rust" />
            <span>
              <b className="font-semibold">{bold}</b> {rest}
            </span>
          </li>
        ))}
      </ul>
      <div className="flex flex-col gap-3 lg:flex-row">
        <Link to="/mine/new" className={cn(buttonStyles.primary, 'hover:text-paper')}>
          <Plus aria-hidden="true" className="size-4" />
          Write an announcement
        </Link>
        <Link to="/mine/events/new" className={cn(buttonStyles.secondary, 'hover:text-ink')}>
          Propose an event
        </Link>
      </div>
    </section>
  )
}
