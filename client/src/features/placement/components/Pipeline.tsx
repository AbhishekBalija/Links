import { cn } from '@/lib/utils'
import type { ApplicantCounts } from '../../jobs/types'
import { pipelineWords } from '../standing'

const segments: { key: keyof ApplicantCounts; color: string }[] = [
  { key: 'applied', color: 'bg-[#cfc4b2]' },
  { key: 'shortlisted', color: 'bg-[#c99a4e]' },
  { key: 'selected', color: 'bg-success' },
  { key: 'rejected', color: 'bg-[#b9798a]' },
]

// Pipeline is an Opportunity's applicants at a glance: the total, the
// statuses in words, and a thin bar of the same numbers.
export function Pipeline({ counts, className }: { counts: ApplicantCounts | undefined; className?: string }) {
  if (!counts) return null
  return (
    <span className={cn('flex flex-col gap-1.5', className)}>
      <span className="flex flex-wrap items-baseline gap-x-2">
        {counts.total > 0 && <span className="font-mono text-lg leading-none">{counts.total}</span>}
        <span className="text-xs text-ink-2">{pipelineWords(counts)}</span>
      </span>
      {counts.total > 0 && (
        <span aria-hidden="true" className="flex h-[5px] gap-0.5 overflow-hidden rounded-full">
          {segments.map(({ key, color }) => (counts[key] > 0 ? <span key={key} className={color} style={{ flex: counts[key] }} /> : null))}
        </span>
      )}
    </span>
  )
}
