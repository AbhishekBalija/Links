import { ChevronLeft } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { ApiRequestError } from '../../../shared/api/types'
import { EmptyState, ErrorState, LoadingStatus, Skeleton } from '../../../shared/ui/states'
import { describeEligibility } from '../../placement/eligibility'
import { dayMonth } from '../../events/standing'
import { Tag } from '../../events/components/Tags'
import { useJob } from '../api'
import { daysLeft, deadlineLine, isUrgent } from '../format'
import { typeLabel, type Opportunity } from '../types'
import { DeadlineTile } from '../components/DeadlineTile'
import { JobAction } from '../components/JobAction'

// JobDetail is one Opportunity: the role, the deadline and who can apply,
// then the one place to act, beside it on desktop and at the bottom on phones.
export default function JobDetail() {
  const { id = '' } = useParams()
  const job = useJob(id)

  return (
    <div className="flex flex-col gap-3 pb-48 lg:gap-5 lg:pb-0">
      <Link to="/jobs" className="-ml-1.5 inline-flex min-h-11 items-center gap-1 self-start rounded-lg px-1.5 text-sm font-semibold">
        <ChevronLeft aria-hidden="true" className="size-5" />
        Jobs
      </Link>
      {job.isPending ? (
        <>
          <LoadingStatus label="Loading the job" />
          <DetailSkeleton />
        </>
      ) : job.isError ? (
        job.error instanceof ApiRequestError && job.error.status === 404 ? (
          <div className="max-w-[640px]">
            <EmptyState title="This opportunity isn't available">
              <p>It may be for other departments or batches, or the link is wrong.</p>
              <Link to="/jobs" className="mt-2 inline-flex min-h-10 items-center text-[15px] font-semibold">
                Back to Jobs
              </Link>
            </EmptyState>
          </div>
        ) : (
          <ErrorState message="This job could not be loaded." onRetry={() => job.refetch()} />
        )
      ) : (
        <div className="grid max-w-[1140px] items-start gap-4 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-6">
          <JobArticle job={job.data} />
          <aside>
            <JobAction job={job.data} />
          </aside>
        </div>
      )}
    </div>
  )
}

function JobArticle({ job }: { job: Opportunity }) {
  const due = deadlineLine(job.apply_by)
  const eligibility = describeEligibility(job.eligibility)
  return (
    <article className="flex flex-col gap-4 lg:gap-5 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:px-9 lg:pt-7 lg:pb-8">
      <header className="flex items-start gap-3.5 lg:gap-5">
        <DeadlineTile iso={job.apply_by} size="lg" />
        <div className="flex flex-col gap-1.5">
          <span className="flex gap-1.5">
            <Tag tone="placement">{typeLabel(job.opportunity_type)}</Tag>
          </span>
          <h1 className="font-serif text-[26px] leading-[1.15] font-medium tracking-[-0.4px] lg:text-[34px] lg:leading-[1.12] lg:tracking-[-0.5px]">{job.title}</h1>
          <span className="font-serif text-lg text-ink-2 lg:text-[21px]">{job.company}</span>
        </div>
      </header>

      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 rounded-xl border border-line bg-surface px-4 py-3.5 text-sm lg:gap-x-[18px] lg:gap-y-2.5 lg:border-0 lg:bg-transparent lg:p-0 lg:text-[15px]">
        <Fact label="Apply by">
          <span className="font-semibold">{due.date} </span>
          <span className="font-mono text-[13px] lg:text-sm">{due.time}</span>
          {job.open && <span className={cn(isUrgent(job.apply_by) && 'font-semibold text-warning')}> · {daysLeft(job.apply_by)}</span>}
        </Fact>
        <Fact label="Who can apply">{eligibility}</Fact>
        <Fact label="How">{job.application_mode === 'internal' ? 'In LINKS' : `On ${job.company}'s site`}</Fact>
        {job.location && <Fact label="Location">{job.location}</Fact>}
        {job.compensation && <Fact label="Pay">{job.compensation}</Fact>}
        <Fact label="Posted by">
          {job.posted_by.full_name}
          {job.published_at && ` · ${dayMonth(job.published_at)}`}
        </Fact>
      </dl>

      {job.description && (
        <div className="flex max-w-[640px] flex-col gap-3 font-serif text-[17px] leading-[1.55] text-prose lg:text-lg lg:leading-[1.6]">
          {job.description.split(/\n\s*\n/).map((paragraph, i) => (
            <p key={i} className="whitespace-pre-line">
              {paragraph}
            </p>
          ))}
        </div>
      )}
    </article>
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

function DetailSkeleton() {
  return (
    <div className="flex max-w-[920px] flex-col gap-4 lg:rounded-xl lg:border lg:border-line lg:bg-surface lg:p-10">
      <div className="flex gap-4">
        <Skeleton className="h-20 w-14 rounded-[10px] lg:w-16" />
        <div className="flex flex-1 flex-col gap-2.5 pt-1">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-7 w-4/5" />
          <Skeleton className="h-5 w-1/3" />
        </div>
      </div>
      <Skeleton className="h-4 w-2/3" />
      <Skeleton className="h-4 w-1/2" />
    </div>
  )
}
