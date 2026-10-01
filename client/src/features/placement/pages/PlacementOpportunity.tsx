import { ChevronLeft, Download } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { buttonStyles } from '../../announcements/buttons'
import { ActionBar } from '../../announcements/components/ActionBar'
import { dayAndDate, dayMonth } from '../../events/standing'
import { Tag } from '../../events/components/Tags'
import { ConfirmDialog } from '../../jobs/components/ConfirmDialog'
import { DeadlineTile } from '../../jobs/components/DeadlineTile'
import { daysLeft, deadlineLine } from '../../jobs/format'
import { typeLabel, type Opportunity } from '../../jobs/types'
import { useDepartments } from '../../announcements/api'
import { useExportApplicants, usePublishing, useManagedOne } from '../api'
import { describeEligibility, eligibilitySummary, fromRules } from '../eligibility'
import { staffStanding } from '../standing'

// PlacementOpportunity is one Opportunity as the office sees it: the details,
// its applicants, and what its status allows (publish a draft, close early,
// edit details).
export default function PlacementOpportunity() {
  const { id = '' } = useParams()
  const item = useManagedOne(id)
  const departments = useDepartments()

  return (
    <div className="flex flex-col gap-3 pb-40 lg:gap-5 lg:pb-0">
      <Link to="/placement" className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5" />
        Placement
      </Link>
      {item.isPending ? (
        <>
          <LoadingStatus label="Loading the opportunity" />
          <Skeleton className="h-72 w-full max-w-[1140px] rounded-xl" />
        </>
      ) : item.isError ? (
        item.error instanceof ApiRequestError && item.error.status === 404 ? (
          <div className="max-w-[640px]">
            <EmptyState title="This opportunity doesn't exist">
              <p>The link may be wrong.</p>
              <Link to="/placement" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
                Back to Placement
              </Link>
            </EmptyState>
          </div>
        ) : (
          <ErrorState message="This opportunity could not be loaded." onRetry={() => item.refetch()} />
        )
      ) : (
        <Page item={item.data} departments={departments.data ?? []} />
      )}
    </div>
  )
}

type Ask = 'publish' | 'close' | null

function Page({ item, departments }: { item: Opportunity; departments: { id: string; code: string }[] }) {
  const [ask, setAsk] = useState<Ask>(null)
  const action = usePublishing()
  const standing = staffStanding(item)
  const open = standing.label === 'Open'
  const choice = fromRules(item.eligibility)
  const who = choice ? eligibilitySummary(choice, departments) : describeEligibility(item.eligibility)
  const pastDeadline = new Date(item.apply_by) <= new Date()

  async function run(kind: 'publish' | 'close') {
    try {
      await action.mutateAsync({ id: item.id, action: kind })
      setAsk(null)
    } catch {
      // The dialog shows the error; the page reloads behind it.
    }
  }

  function close() {
    setAsk(null)
    action.reset()
  }

  const error = action.error
    ? action.error instanceof ApiRequestError && action.error.status === 409
      ? 'Someone in the office changed it a moment ago. The page now shows where it stands.'
      : action.error instanceof ApiRequestError && action.error.status === 400
        ? 'Its apply-by date has passed. Edit it to a time after now, then publish.'
        : 'That did not go through. Try again.'
    : null

  const note =
    item.status === 'draft'
      ? 'Only the placement office, the principal and admins can see it.'
      : open
        ? 'Details and the deadline can still change.'
        : item.status === 'published'
          ? `The deadline passed on ${dayMonth(item.apply_by)}, so students can't apply any more.`
          : standing.label === 'Closed early'
            ? `Closed early on ${dayAndDate(item.closed_at ?? item.updated_at)}. It can't be reopened.`
            : `Closed on ${dayAndDate(item.closed_at ?? item.apply_by)}.`

  return (
    <>
      <div className="grid max-w-[1140px] items-start gap-4 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-6">
        <div className="flex flex-col gap-4">
          <Details item={item} who={who} standingLabel={standing.label} standingTone={standing.tone} />
          <ActionBar
            sentence={<span>{note}</span>}
            start={open ? <button type="button" onClick={() => setAsk('close')} className={buttonStyles.secondary}>Close early</button> : undefined}
          >
            {(item.status === 'draft' || open) && (
              <Link to={`/placement/${item.id}/edit`} className={buttonStyles.secondary}>
                {item.status === 'draft' ? 'Edit' : 'Edit details'}
              </Link>
            )}
            {item.status === 'draft' && (
              <button type="button" onClick={() => setAsk('publish')} className={buttonStyles.primary}>
                Publish
              </button>
            )}
          </ActionBar>
        </div>
        {item.status !== 'draft' && (
          <aside>
            <ApplicantsPanel item={item} />
          </aside>
        )}
      </div>

      <ConfirmDialog
        open={ask === 'publish'}
        title={`Publish to ${who}?`}
        cancelLabel="Keep as a draft"
        confirmLabel="Publish"
        busyLabel="Publishing…"
        busy={action.isPending}
        error={error}
        onCancel={close}
        onConfirm={() => run('publish')}
      >
        {pastDeadline ? (
          <p className="rounded-lg bg-warning-soft px-3 py-2.5 text-sm text-warning-ink">Its apply-by date has passed. Edit it first, then publish.</p>
        ) : (
          <p>
            It shows in their Jobs right away, open until <b className="font-semibold text-ink">{deadlineLine(item.apply_by).date}, {deadlineLine(item.apply_by).time}</b>. After publishing you can edit the details and move the deadline later, but not change how students apply.
          </p>
        )}
      </ConfirmDialog>

      <ConfirmDialog
        open={ask === 'close'}
        title="Close applications now?"
        cancelLabel="Keep open"
        confirmLabel="Close applications"
        busyLabel="Closing…"
        busy={action.isPending}
        danger
        error={error}
        onCancel={close}
        onConfirm={() => run('close')}
      >
        <p>Students can't apply after this, and it can't be reopened. Every application stays, and you can keep shortlisting and exporting.</p>
      </ConfirmDialog>
    </>
  )
}

function Details({ item, who, standingLabel, standingTone }: { item: Opportunity; who: string; standingLabel: string; standingTone: 'plain' | 'going' }) {
  const due = deadlineLine(item.apply_by)
  const open = standingLabel === 'Open'
  return (
    <article className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
      <div className="flex items-center gap-2">
        <Tag tone={standingTone}>{standingLabel}</Tag>
        <span className="text-[13px] text-ink-3">
          {item.published_at ? `Published ${dayMonth(item.published_at)} by ${item.posted_by.full_name}` : `Saved by ${item.posted_by.full_name}`}
        </span>
      </div>
      <header className="flex items-start gap-3.5 lg:gap-5">
        <DeadlineTile iso={item.apply_by} size="lg" />
        <div className="flex flex-col gap-1.5">
          <span className="flex">
            <Tag tone="placement">{typeLabel(item.opportunity_type)}</Tag>
          </span>
          <h1 className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] lg:text-[34px] lg:leading-[1.12] lg:tracking-[-0.5px]">{item.title}</h1>
          <span className="font-serif text-lg text-ink-2 lg:text-[21px]">{item.company}</span>
        </div>
      </header>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 rounded-xl border border-line bg-surface px-4 py-3.5 text-sm lg:gap-x-[18px] lg:gap-y-2.5 lg:border-0 lg:bg-transparent lg:p-0 lg:text-[15px]">
        <Fact label="Apply by">
          <span className="font-semibold">{due.date} </span>
          <span className="font-mono text-[13px] lg:text-sm">{due.time}</span>
          {open && ` · ${daysLeft(item.apply_by)}`}
        </Fact>
        <Fact label="Who can apply">{who.charAt(0).toUpperCase() + who.slice(1)}</Fact>
        <Fact label="How students apply">
          {item.application_mode === 'internal' ? (
            'In LINKS'
          ) : (
            <>
              On the company's site ·{' '}
              <a href={item.external_url ?? '#'} target="_blank" rel="noopener noreferrer" className="break-all font-semibold">
                {item.external_url}
              </a>
            </>
          )}
        </Fact>
        {(item.location || item.compensation) && <Fact label="Location and pay">{[item.location, item.compensation].filter(Boolean).join(' · ')}</Fact>}
      </dl>
      {item.description && (
        <div className="flex max-w-[640px] flex-col gap-3 font-serif text-[17px] leading-[1.55] text-prose lg:text-lg">
          {item.description.split(/\n\s*\n/).map((paragraph, i) => (
            <p key={i} className="whitespace-pre-line">
              {paragraph}
            </p>
          ))}
        </div>
      )}
    </article>
  )
}

function ApplicantsPanel({ item }: { item: Opportunity }) {
  const counts = item.applicant_counts
  const exporter = useExportApplicants(item)
  if (!counts) return null
  const cells: [number, string, boolean?][] = [
    [counts.applied, 'to review'],
    [counts.shortlisted, 'shortlisted'],
    [counts.selected, 'selected'],
    [counts.rejected, 'rejected'],
    [counts.withdrawn, 'withdrew', true],
  ]
  return (
    <section aria-labelledby="applicants-h" className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4 lg:p-5">
      <h2 id="applicants-h" className="font-serif text-[21px] font-medium">
        Applicants
      </h2>
      {counts.total === 0 && counts.withdrawn === 0 ? (
        <p className="text-sm text-ink-2">No one has applied yet.</p>
      ) : (
        <div className="grid grid-cols-3 gap-3.5">
          {cells.map(([n, word, muted]) => (
            <span key={word} className="flex flex-col">
              <span className={cn('font-mono text-[28px] leading-tight', muted && 'text-ink-3')}>{n}</span>
              <span className="text-[13px] text-ink-2">{word}</span>
            </span>
          ))}
        </div>
      )}
      {(counts.total > 0 || counts.withdrawn > 0) && (
        <>
          <Link to={`/placement/${item.id}/applicants`} className={cn(buttonStyles.primary, 'min-h-12 w-full hover:text-paper lg:flex-none')}>
            Open applicant list
          </Link>
          <button type="button" onClick={() => exporter.mutate(null)} disabled={exporter.isPending} className={cn(buttonStyles.secondary, 'self-start')}>
            <Download aria-hidden="true" className="size-4" />
            {exporter.isPending ? 'Downloading…' : 'Export CSV'}
          </button>
          <p className="text-xs text-ink-3">The CSV includes emails and USNs, never phone numbers, so each download is logged.</p>
          {exporter.isError && (
            <p role="alert" className="text-xs font-semibold text-danger">
              The download didn't work. Try again.
            </p>
          )}
        </>
      )}
    </section>
  )
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-ink-3">{label}</dt>
      <dd>{children}</dd>
    </>
  )
}
