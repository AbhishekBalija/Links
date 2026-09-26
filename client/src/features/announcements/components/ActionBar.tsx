import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

type Props = {
  // What happens, in words, next to the buttons that do it.
  sentence: ReactNode
  // A rarer action kept apart from the main ones (e.g. Withdraw).
  start?: ReactNode
  children?: ReactNode
  // danger: a destructive confirmation; confirm: a consequential one.
  tone?: 'normal' | 'danger' | 'confirm'
  labelledBy?: string
  // stacked puts the sentence (e.g. a form) above the buttons on desktop too.
  stacked?: boolean
  // Escape inside the bar backs out of a confirmation.
  onEscape?: () => void
}

// ActionBar is the one place a screen's actions live: pinned to the bottom
// on phones, a panel under the content on desktop. On phones the sentence
// hides while a text field has focus, so the keyboard doesn't cover the
// field being typed in.
export function ActionBar({ sentence, start, children, tone = 'normal', labelledBy, stacked, onEscape }: Props) {
  return (
    <footer
      role={tone === 'normal' ? undefined : 'alertdialog'}
      aria-labelledby={labelledBy}
      onKeyDown={(e) => {
        if (e.key === 'Escape' && onEscape) {
          e.preventDefault()
          onEscape()
        }
      }}
      className={cn(
        'fixed inset-x-0 bottom-0 z-40 flex flex-col gap-2.5 px-4 pt-3 pb-[max(env(safe-area-inset-bottom),16px)]',
        'lg:sticky lg:bottom-4 lg:mt-2 lg:gap-6 lg:rounded-xl lg:py-3.5 lg:pr-4 lg:pl-6',
        !stacked && 'lg:flex-row lg:items-center lg:justify-between',
        tone === 'danger' && 'bg-danger-soft shadow-[0_-1px_0_#e8c9d2] lg:shadow-none',
        tone === 'confirm' && 'bg-surface shadow-[0_-1.5px_0_var(--color-ink)] lg:border-[1.5px] lg:border-ink lg:shadow-none',
        tone === 'normal' && 'bg-surface shadow-[0_-1px_0_var(--color-line)] lg:border lg:border-line lg:shadow-none',
      )}
    >
      {start && <div className="hidden lg:flex lg:items-center lg:gap-3">{start}</div>}
      <div className={cn('flex flex-col gap-2.5', stacked ? 'lg:gap-3' : 'lg:ml-auto lg:flex-row lg:items-center lg:gap-4')}>
        <div
          className={cn(
            'text-[13px] leading-snug lg:text-[15px]',
            !stacked && 'lg:text-right',
            tone === 'danger' ? 'text-danger-ink' : 'text-ink-2',
            !stacked && 'max-lg:group-has-[:is(input,textarea):focus]/page:hidden',
          )}
        >
          {sentence}
        </div>
        {(children || start) && (
          <div className={cn('flex gap-2.5', stacked && 'lg:justify-end')}>
            {start && <div className="flex lg:hidden">{start}</div>}
            {children}
          </div>
        )}
      </div>
    </footer>
  )
}
