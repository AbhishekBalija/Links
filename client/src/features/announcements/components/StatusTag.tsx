import { cn } from '@/lib/utils'
import type { Tone } from '../status'

const tones: Record<Tone, string> = {
  neutral: 'bg-well text-ink-2',
  live: 'bg-success-soft text-success-ink',
  danger: 'bg-danger-soft text-danger-ink',
}

export function StatusTag({ text, tone }: { text: string; tone: Tone }) {
  return (
    <span className={cn('inline-block rounded px-2 py-0.5 text-xs font-semibold whitespace-nowrap', tones[tone])}>{text}</span>
  )
}
