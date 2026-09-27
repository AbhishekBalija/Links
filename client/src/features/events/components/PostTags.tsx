import { cn } from '@/lib/utils'
import type { ProposalStanding, ProposalTone } from '../standing'

// PostTag says what kind of post a row in My posts is.
export function PostTag({ kind }: { kind: 'event' | 'announcement' }) {
  return (
    <span
      className={cn(
        'inline-block rounded px-[7px] py-0.5 text-[11px] leading-4 font-semibold whitespace-nowrap',
        kind === 'event' ? 'bg-surface text-ink shadow-[inset_0_0_0_1px_#b9ae9c]' : 'bg-well text-ink-2',
      )}
    >
      {kind === 'event' ? 'Event' : 'Announcement'}
    </span>
  )
}

const tones: Record<ProposalTone, string> = {
  neutral: 'bg-well text-ink-2',
  warning: 'bg-warning-soft text-warning-ink',
  live: 'bg-success-soft text-success-ink',
  danger: 'bg-danger-soft text-danger-ink',
}

export function StandingTag({ standing }: { standing: Pick<ProposalStanding, 'tag' | 'tone'> }) {
  return <span className={cn('inline-block rounded px-[7px] py-0.5 text-[11px] leading-4 font-semibold whitespace-nowrap', tones[standing.tone])}>{standing.tag}</span>
}
