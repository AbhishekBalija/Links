import { cn } from '@/lib/utils'
import { dateParts } from '../format'

// DateTile is an Event's date at a glance, like a desk calendar: month,
// day and weekday. Screen readers get the date from the row's text instead.
export function DateTile({ iso, size = 'md' }: { iso: string; size?: 'sm' | 'md' | 'lg' }) {
  const { month, day, weekday } = dateParts(iso)
  return (
    <span
      aria-hidden="true"
      className={cn(
        'flex shrink-0 flex-col items-center gap-px rounded-[10px] bg-paper pt-[7px] pb-2 text-ink-2',
        size === 'sm' && 'w-12',
        size === 'md' && 'w-[52px] lg:w-[60px]',
        size === 'lg' && 'w-14 lg:w-[72px]',
      )}
    >
      <span className="font-mono text-[11px] tracking-[1px]">{month}</span>
      <span className={cn('font-serif leading-none font-medium text-ink', size === 'lg' ? 'text-[26px] lg:text-[30px]' : 'text-[22px] lg:text-[26px]')}>{day}</span>
      <span className="font-mono text-[11px] tracking-[1px]">{weekday}</span>
    </span>
  )
}
