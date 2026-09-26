import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { CategoryTag } from '../../../notices/components/CategoryTag'
import { audienceLabel } from '../../../notices/format'
import { hasExpired, waited } from '../../status'
import type { QueueItem } from '../../types'
import { Chip } from './Chip'

// QueueRow is one waiting item in the queue's list.
export function QueueRow({ item, selected }: { item: QueueItem; selected: boolean }) {
  const wait = waited(item.submitted_at)
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
            <CategoryTag category={item.category} />
            {item.kind === 'edit' && <Chip>Edit</Chip>}
            {hasExpired(item) && <Chip tone="warning">Expired</Chip>}
          </span>
          <span className={cn('font-mono text-xs whitespace-nowrap', wait.long ? 'text-warning' : 'text-ink-3')}>{wait.text.replace('waiting ', '')}</span>
        </span>
        <span className={cn('text-base leading-snug lg:text-[15px]', selected ? 'font-semibold' : 'font-semibold lg:font-medium')}>{item.title}</span>
        <span className="text-[13px] text-ink-3">
          {item.publisher_name} · {audienceLabel(item.audience)}
        </span>
      </span>
    </Link>
  )
}
