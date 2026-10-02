import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { waited } from '../../announcements/status'
import { Chip } from '../../announcements/components/queue/Chip'
import type { AccessRequest } from '../types'

// RequestRow is one waiting Access request in the list.
export function RequestRow({ request, selected, to }: { request: AccessRequest; selected: boolean; to: string }) {
  const id = request.student_identity
  return (
    <Link
      to={to}
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
          <span className={cn('text-base leading-snug lg:text-[15px]', selected ? 'font-semibold' : 'font-semibold lg:font-medium')}>
            {request.profile?.full_name ?? request.email}
          </span>
          <span className="font-mono text-xs whitespace-nowrap text-ink-3">{waited(request.created_at).text}</span>
        </span>
        {id && (
          <span className="text-[13px] text-ink-3">
            <span className="font-mono text-xs text-ink-2">{id.usn}</span> · {id.department_code} · Batch {id.batch_year}
          </span>
        )}
        {request.reported_at && (
          <span className="self-start">
            <Chip tone="warning">Reported on first sign-in</Chip>
          </span>
        )}
      </span>
    </Link>
  )
}
