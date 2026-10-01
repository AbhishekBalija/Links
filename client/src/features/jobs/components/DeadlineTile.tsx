import { cn } from '@/lib/utils'
import { deadlineParts } from '../format'

// DeadlineTile is the apply-by date at a glance. It has the shape of the
// event date tile but the placement tint, so a deadline never reads as an
// event day. Screen readers get the date from the row's text instead.
export function DeadlineTile({ iso, size = 'md' }: { iso: string; size?: 'md' | 'lg' }) {
  const { day, month } = deadlineParts(iso)
  return (
    <span
      aria-hidden="true"
      className={cn(
        'flex shrink-0 flex-col items-center gap-px rounded-[10px] bg-tag-placement pt-[7px] pb-2 text-tag-placement-ink',
        size === 'md' ? 'w-[52px] lg:w-[60px]' : 'w-14 lg:w-16',
      )}
    >
      <span className="font-mono text-[10px] tracking-[1px]">BY</span>
      <span className="font-serif text-[22px] leading-none font-medium text-ink lg:text-[26px]">{day}</span>
      <span className="font-mono text-[11px] tracking-[1px]">{month}</span>
    </span>
  )
}
