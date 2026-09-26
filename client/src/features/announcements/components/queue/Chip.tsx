import { cn } from '@/lib/utils'

// Chip marks a queue item as an edit or as expired.
export function Chip({ children, tone = 'neutral' }: { children: string; tone?: 'neutral' | 'warning' }) {
  return (
    <span className={cn('rounded px-1.5 py-0.5 text-[11px] leading-4 font-semibold', tone === 'warning' ? 'bg-warning-soft text-warning-ink' : 'bg-well text-ink-2')}>
      {children}
    </span>
  )
}
