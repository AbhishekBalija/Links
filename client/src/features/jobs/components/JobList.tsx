import { Skeleton } from '../../../shared/ui/states'
import { jobGroup } from '../format'
import type { Opportunity } from '../types'
import { JobRow } from './JobRow'

type Group = { label: string | null; jobs: Opportunity[] }

// Open jobs come under "Closing this week" and "Later"; the applied and
// closed views are one plain list.
function groups(jobs: Opportunity[], grouped: boolean, now: Date): Group[] {
  if (!grouped) return [{ label: null, jobs }]
  const out: Group[] = []
  for (const job of jobs) {
    const label = jobGroup(job.apply_by, now)
    const last = out[out.length - 1]
    if (last && last.label === label) last.jobs.push(job)
    else out.push({ label, jobs: [job] })
  }
  return out
}

const panel =
  'flex flex-col overflow-hidden rounded-xl border border-line bg-surface lg:gap-0.5 lg:p-1.5 [&>li+li]:shadow-[0_-1px_0_#efe9de] lg:[&>li+li]:shadow-none'

export function JobList({ jobs, grouped }: { jobs: Opportunity[]; grouped: boolean }) {
  const now = new Date()
  return (
    <div className="flex flex-col gap-4 lg:gap-[18px]">
      {groups(jobs, grouped, now).map((group) =>
        group.label ? (
          <section key={group.label} aria-label={group.label} className="flex flex-col gap-1.5">
            <h2 className="mx-1 font-serif text-xl font-medium text-ink-3 lg:text-[22px]">{group.label}</h2>
            <ul className={panel}>
              {group.jobs.map((job) => (
                <JobRow key={job.id} job={job} />
              ))}
            </ul>
          </section>
        ) : (
          <ul key="all" className={panel}>
            {group.jobs.map((job) => (
              <JobRow key={job.id} job={job} />
            ))}
          </ul>
        ),
      )}
    </div>
  )
}

export function JobListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <ul aria-hidden="true" className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface">
      {Array.from({ length: rows }, (_, i) => (
        <li key={i} className="flex gap-3 px-3.5 py-3">
          <Skeleton className="h-16 w-[52px] rounded-[10px]" />
          <span className="flex flex-1 flex-col gap-2 pt-1">
            <Skeleton className="h-4 w-4/5" />
            <Skeleton className="h-3 w-1/2" />
            <Skeleton className="h-3 w-1/4" />
          </span>
        </li>
      ))}
    </ul>
  )
}
