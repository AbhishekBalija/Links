import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Chip } from '../../announcements/components/queue/Chip'
import { waited } from '../../announcements/status'
import { audienceLabel } from '../../notices/format'
import type { EventQueueItem } from '../types'
import { PostTag } from './PostTags'

// EventQueueRow is one waiting event proposal in the approval queue's list.
export function EventQueueRow({ item, selected }: { item: EventQueueItem; selected: boolean }) {
  const wait = waited(item.submitted_at ?? item.updated_at)
  const audience = audienceLabel(
    (item.audience ?? []).map((r) => ({ department_id: r.department_id ?? null, department_code: r.department_code ?? null, batch_year: r.batch_year ?? null, role: r.role ?? null })),
  )
  return (
    <Link
      to={`/approvals/${item.id}`}
      replace
      aria-current={selected ? 'true' : undefined}
      className={cn(
        'flex gap-2.5 rounded-[10px] border border-line bg-surface px-4 py-3.5 text-ink hover:text-ink',
        'lg:rounded-lg lg:border-0 lg:bg-transparent lg:py-3 lg:pr-3.5 lg:pl-2.5 lg:hover:bg-paper',
        selected && 'lg:bg-paper',
      )}
    >
      <span aria-hidden="true" className={cn('mt-2 hidden size-1.5 shrink-0 rounded-full lg:block', selected && 'bg-rust')} />
      <span className="flex min-w-0 flex-1 flex-col gap-1.5">
        <span className="flex items-center justify-between gap-2">
          <span className="flex gap-1.5">
            <PostTag kind="event" />
            {item.stage === 'final' && <Chip>Final</Chip>}
          </span>
          <span className={cn('font-mono text-xs whitespace-nowrap', wait.long ? 'text-warning' : 'text-ink-3')}>{wait.text.replace('waiting ', '')}</span>
        </span>
        <span className={cn('text-base leading-snug lg:text-[15px]', selected ? 'font-semibold' : 'font-semibold lg:font-medium')}>{item.title}</span>
        <span className="text-[13px] text-ink-3">
          {item.proposer_name} · {audience}
          {item.stage === 'final' && item.reviews?.some((r) => r.stage === 'hod' && r.decision === 'approve') && ' · HOD approved'}
        </span>
      </span>
    </Link>
  )
}
