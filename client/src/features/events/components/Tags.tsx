import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

const tones = {
  plain: 'bg-well text-ink-2',
  department: 'bg-tag-department text-tag-department-ink',
  going: 'bg-success-soft text-success-ink',
  cancelled: 'bg-danger-soft text-danger-ink',
  warning: 'bg-warning-soft text-warning-ink',
  placement: 'bg-tag-placement text-tag-placement-ink',
  outline: 'text-ink-2 shadow-[inset_0_0_0_1px_#d6ccbb]',
}

export function Tag({ tone = 'plain', children }: { tone?: keyof typeof tones; children: ReactNode }) {
  return <span className={cn('inline-block rounded px-[7px] py-0.5 text-[11px] leading-4 font-semibold whitespace-nowrap', tones[tone])}>{children}</span>
}
