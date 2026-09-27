import { cn } from '@/lib/utils'
import type { Answer } from '../types'

const options: { value: Answer; label: string }[] = [
  { value: 'going', label: 'Going' },
  { value: 'interested', label: 'Interested' },
  { value: 'not_going', label: "Can't go" },
]

type Props = {
  value: Answer | null
  // When the event is full, Going can't be chosen unless it already is.
  full: boolean
  busy: boolean
  onAnswer: (answer: Answer) => void
  grow?: boolean
}

// AnswerControl is the one way to answer an Event, on every device: three
// joined options, the chosen one filled.
export function AnswerControl({ value, full, busy, onAnswer, grow = false }: Props) {
  return (
    <div role="group" aria-label="Your answer" className={cn('flex gap-1 rounded-[10px] bg-well p-1', grow && 'w-full')}>
      {options.map((option) => {
        const chosen = option.value === value
        const blocked = option.value === 'going' && full && !chosen
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={chosen}
            disabled={busy || blocked}
            onClick={() => !chosen && onAnswer(option.value)}
            className={cn(
              'min-h-11 rounded-[7px] px-[18px] text-[15px] font-semibold whitespace-nowrap',
              grow && 'flex-1 px-2',
              chosen ? 'bg-ink text-paper' : 'text-ink hover:bg-surface/70',
              blocked && 'text-ink-3 hover:bg-transparent',
              busy && !chosen && 'opacity-60',
            )}
          >
            {blocked ? 'Going · full' : option.label}
          </button>
        )
      })}
    </div>
  )
}
