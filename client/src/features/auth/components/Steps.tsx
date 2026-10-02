import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'

export type Step = { label: string; detail: string; state: 'done' | 'now' | 'later' }

// Steps shows where an Access request stands: sent, the HOD decides, you
// sign in.
export function Steps({ steps }: { steps: Step[] }) {
  return (
    <ol className="flex flex-col gap-4 rounded-xl border border-line bg-surface p-[18px]">
      {steps.map((step) => (
        <li key={step.label} className="flex items-start gap-3.5" aria-current={step.state === 'now' ? 'step' : undefined}>
          {step.state === 'done' ? (
            <span className="flex size-6 items-center justify-center rounded-full bg-success text-surface">
              <Check aria-hidden="true" className="size-4" />
            </span>
          ) : step.state === 'now' ? (
            <span className="flex size-6 items-center justify-center rounded-full border-2 border-rust">
              <span className="size-2 rounded-full bg-rust" />
            </span>
          ) : (
            <span className="size-6 rounded-full border-2 border-line" />
          )}
          <span className="flex flex-col gap-0.5 pt-0.5">
            <span className={cn('text-[15px]', step.state === 'later' ? 'font-medium text-ink-3' : 'font-semibold')}>{step.label}</span>
            <span className="text-[13px] leading-snug text-ink-3">{step.detail}</span>
          </span>
        </li>
      ))}
    </ol>
  )
}
