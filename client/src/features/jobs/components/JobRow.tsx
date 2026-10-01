import { ExternalLink } from 'lucide-react'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Tag } from '../../events/components/Tags'
import { daysLeft, isUrgent, jobMeta } from '../format'
import { studentStatus } from '../standing'
import { typeLabel, type Opportunity } from '../types'
import { DeadlineTile } from './DeadlineTile'

// Where the student stands, or how to apply when they haven't yet. Once they
// applied, the countdown is gone: it would only nag.
function Standing({ job }: { job: Opportunity }) {
  if (job.my_application) {
    const status = studentStatus(job.my_application)
    return <Tag tone={status.tone}>{status.label}</Tag>
  }
  return (
    <>
      {job.application_mode === 'external' && job.open && (
        <span className="inline-flex items-center gap-1 text-xs whitespace-nowrap text-ink-2">
          <ExternalLink aria-hidden="true" className="size-3.5" />
          Company site
        </span>
      )}
      <span className={cn('font-mono text-[11px] whitespace-nowrap lg:text-xs', isUrgent(job.apply_by) ? 'text-warning' : 'text-ink-3')}>
        {daysLeft(job.apply_by)}
      </span>
    </>
  )
}

// JobRow is one Opportunity in a list. The whole row opens it; the deadline
// and the company are what students scan for.
export function JobRow({ job }: { job: Opportunity }) {
  return (
    <li>
      <Link
        to={`/jobs/${job.id}`}
        className="flex gap-3 px-3.5 py-3 text-ink hover:bg-paper hover:text-ink lg:items-center lg:gap-[18px] lg:rounded-[10px] lg:py-3 lg:pr-4 lg:pl-3"
      >
        <DeadlineTile iso={job.apply_by} />
        <span className="flex min-w-0 flex-1 flex-col gap-1 lg:gap-[5px]">
          <span className="order-3 flex flex-wrap items-center gap-1.5 lg:order-none">
            <Tag tone="placement">{typeLabel(job.opportunity_type)}</Tag>
            <span className="flex items-center gap-1.5 lg:hidden">
              <Standing job={job} />
            </span>
          </span>
          <span className="text-base leading-[1.3] font-semibold lg:text-[17px]">{job.title}</span>
          <span className="text-[13px] text-ink-3 lg:text-ink-2">{jobMeta(job)}</span>
        </span>
        <span className="hidden shrink-0 items-center gap-2.5 lg:flex">
          <Standing job={job} />
        </span>
      </Link>
    </li>
  )
}
