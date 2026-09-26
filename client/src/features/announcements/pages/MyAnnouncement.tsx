import { ChevronLeft } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { useDepartments, usePreview, useAuthored, useWithdraw } from '../api'
import { CategoryTag } from '../../notices/components/CategoryTag'
import { fullDate } from '../../notices/format'
import { describe, toRules } from '../audience'
import { buttonStyles } from '../buttons'
import { ActionBar } from '../components/ActionBar'
import { StatusTag } from '../components/StatusTag'
import { approverPhrase, standing } from '../status'
import type { Authored } from '../types'

// MyAnnouncement is one of the author's own notices, read-only, with its
// actions in one bar at the bottom.
export default function MyAnnouncement() {
  const { id = '' } = useParams()
  const item = useAuthored(id)
  const departments = useDepartments()

  return (
    <div className="group/page flex flex-col gap-4 pb-40 lg:gap-5 lg:pb-0">
      <Link to="/mine" className="-ml-2 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-2 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5 lg:size-4" />
        My announcements
      </Link>
      {item.isPending ? (
        <ViewSkeleton />
      ) : item.isError ? (
        item.error instanceof ApiRequestError && item.error.status === 404 ? (
          <EmptyState title="This announcement isn't available">It may have been removed, or it isn't yours.</EmptyState>
        ) : (
          <ErrorState message="This announcement could not be loaded." onRetry={() => item.refetch()} />
        )
      ) : (
        <View item={item.data} codes={new Map((departments.data ?? []).map((d) => [d.id, d.code]))} />
      )}
    </div>
  )
}

function View({ item, codes }: { item: Authored; codes: Map<string, string> }) {
  const status = standing(item)
  const audience = toRules(item.audience)
  const preview = usePreview(item.category, audience)
  const paragraphs = item.body.split(/\n\s*\n/)
  const posted = item.published_at ?? item.created_at

  return (
    <>
      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_300px] lg:gap-7">
        <article className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-10 lg:pt-8 lg:pb-9">
          <div className="flex flex-wrap items-center gap-2.5 text-xs">
            <CategoryTag category={item.category} className="lg:text-xs" />
            <span className="lg:hidden">
              <StatusTag text={status.tag} tone={status.tone} />
            </span>
            <span className="text-ink-3">{status.when}</span>
          </div>
          <h1 className="font-serif text-[27px] leading-[1.15] font-medium tracking-[-0.4px] lg:text-[36px] lg:leading-[1.12] lg:tracking-[-0.5px]">
            {item.title}
          </h1>
          <p className="text-[13px] text-ink-2 lg:hidden">
            To <b className="font-semibold text-ink">{describe(audience, codes)}</b>
            {preview.data && ` · about ${preview.data.reach.toLocaleString('en-IN')} people`}
          </p>
          <div className="flex max-w-[660px] flex-col gap-3 font-serif text-[17px] leading-relaxed text-prose lg:gap-3.5 lg:text-[19px]">
            {paragraphs.map((text, i) => (
              <p key={i} className="whitespace-pre-line">
                {text}
              </p>
            ))}
          </div>
        </article>
        <aside className="hidden flex-col gap-4 rounded-xl border border-line bg-surface p-5 lg:flex">
          <div className="flex flex-col items-start gap-1.5">
            <span className="text-xs text-ink-3">Status</span>
            <StatusTag text={status.tag} tone={status.tone} />
          </div>
          <div className="flex flex-col gap-1">
            <span className="text-xs text-ink-3">Sent to</span>
            <span className="text-sm font-semibold">{describe(audience, codes)}</span>
            {preview.data && <span className="font-mono text-xs text-ink-3">about {preview.data.reach.toLocaleString('en-IN')} people</span>}
          </div>
          <div className="flex flex-col gap-1">
            <span className="text-xs text-ink-3">{item.status === 'published' ? 'Published' : 'Created'}</span>
            <span className="font-mono text-sm">{fullDate(posted)}</span>
          </div>
          {item.expires_at && (
            <div className="flex flex-col gap-1">
              <span className="text-xs text-ink-3">{status.expired ? 'Expired' : 'Expires'}</span>
              <span className="font-mono text-sm">{fullDate(item.expires_at)}</span>
            </div>
          )}
        </aside>
      </div>
      <Bar item={item} publishesDirectly={preview.data?.publishes_directly} />
    </>
  )
}

// Bar holds the actions for each status, and says why when there are none.
function Bar({ item, publishesDirectly }: { item: Authored; publishesDirectly: boolean | undefined }) {
  const navigate = useNavigate()
  const withdraw = useWithdraw()
  const [confirming, setConfirming] = useState(false)
  const [failed, setFailed] = useState('')
  const status = standing(item)
  const edit = (label: string, primary = true) => (
    <Link to={`/mine/${item.id}/edit`} className={primary ? buttonStyles.primary : buttonStyles.secondary}>
      {label}
    </Link>
  )

  if (confirming) {
    return (
      <ActionBar
        tone="danger"
        labelledBy="withdraw-h"
        sentence={
          <span id="withdraw-h">
            <b className="font-semibold">Take this down for everyone?</b> It leaves every feed and can't be put back.
            {failed && <span className="block font-semibold">{failed}</span>}
          </span>
        }
      >
        <button type="button" onClick={() => setConfirming(false)} className={buttonStyles.secondary + ' flex-1 bg-surface lg:flex-none'}>
          Keep it
        </button>
        <button
          type="button"
          disabled={withdraw.isPending}
          onClick={async () => {
            setFailed('')
            try {
              await withdraw.mutateAsync(item.id)
              navigate('/mine?status=ended', { replace: true })
            } catch {
              setFailed("It couldn't be withdrawn. Try again.")
            }
          }}
          className={buttonStyles.danger}
        >
          {withdraw.isPending ? 'Withdrawing…' : 'Withdraw'}
        </button>
      </ActionBar>
    )
  }

  const withdrawButton = (
    <button type="button" onClick={() => setConfirming(true)} className={buttonStyles.secondary}>
      Withdraw
    </button>
  )

  switch (item.status) {
    case 'draft':
      return <ActionBar sentence="A draft. Nobody else can see it yet.">{edit('Continue editing')}</ActionBar>
    case 'rejected':
      return (
        <ActionBar sentence={<><b className="font-semibold text-ink">{item.approver ?? 'The approver'} sent this back:</b> {item.review_note}</>}>
          {edit('Edit and resubmit')}
        </ActionBar>
      )
    case 'pending':
      return (
        <ActionBar sentence={<>With <b className="font-semibold text-ink">{approverPhrase(item.approver)}</b>. You can edit it once they decide.</>} />
      )
    case 'withdrawn':
      return <ActionBar sentence="Withdrawn. Readers no longer see it." />
    case 'published': {
      if (status.expired) {
        return <ActionBar sentence="Expired. Readers no longer see it." />
      }
      if (item.edit?.status === 'pending') {
        return (
          <ActionBar
            start={withdrawButton}
            sentence={<>Your edit is with <b className="font-semibold text-ink">{approverPhrase(item.edit.approver)}</b>. Readers see this version until it's approved.</>}
          />
        )
      }
      if (item.edit?.status === 'rejected') {
        return (
          <ActionBar start={withdrawButton} sentence={<><b className="font-semibold text-ink">Your edit was sent back:</b> {item.edit.review_note} Readers still see this version.</>}>
            {edit('Edit again')}
          </ActionBar>
        )
      }
      return (
        <ActionBar
          start={withdrawButton}
          sentence={publishesDirectly ? 'Edits go live straight away.' : 'Editing sends it for approval again. Readers keep this version meanwhile.'}
        >
          {edit('Edit')}
        </ActionBar>
      )
    }
  }
}

function ViewSkeleton() {
  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_300px] lg:gap-7">
      <LoadingStatus label="Loading the announcement" />
      <div className="flex flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-10">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-9 w-4/5" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
      </div>
    </div>
  )
}
