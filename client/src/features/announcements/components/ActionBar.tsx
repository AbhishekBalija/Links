import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

type Props = {
  // What happens, in words, next to the buttons that do it.
  sentence: ReactNode
  // A rarer action kept apart from the main ones (e.g. Withdraw).
  start?: ReactNode
  children?: ReactNode
  tone?: 'normal' | 'danger'
  labelledBy?: string
}

// ActionBar is the one place a screen's actions live: pinned to the bottom
// on phones, a panel under the content on desktop. On phones the sentence
// hides while a text field has focus, so the keyboard doesn't cover the
// field being typed in.
export function ActionBar({ sentence, start, children, tone = 'normal', labelledBy }: Props) {
  return (
    <footer
      role={tone === 'danger' ? 'alertdialog' : undefined}
      aria-labelledby={labelledBy}
      className={cn(
        'fixed inset-x-0 bottom-0 z-40 flex flex-col gap-2.5 px-4 pt-3 pb-[max(env(safe-area-inset-bottom),16px)]',
        'lg:sticky lg:bottom-4 lg:mt-2 lg:flex-row lg:items-center lg:justify-between lg:gap-6 lg:rounded-xl lg:py-3.5 lg:pr-4 lg:pl-6',
        tone === 'danger'
          ? 'bg-danger-soft shadow-[0_-1px_0_#e8c9d2] lg:shadow-none'
          : 'bg-surface shadow-[0_-1px_0_var(--color-line)] lg:border lg:border-line lg:shadow-none',
      )}
    >
      {start && <div className="hidden lg:flex lg:items-center lg:gap-3">{start}</div>}
      <div className="flex flex-col gap-2.5 lg:ml-auto lg:flex-row lg:items-center lg:gap-4">
        <div
          className={cn(
            'text-[13px] leading-snug lg:text-right lg:text-[15px]',
            tone === 'danger' ? 'text-danger-ink' : 'text-ink-2',
            'max-lg:group-has-[:is(input,textarea):focus]/page:hidden',
          )}
        >
          {sentence}
        </div>
        {(children || start) && (
          <div className="flex gap-2.5">
            {start && <div className="flex lg:hidden">{start}</div>}
            {children}
          </div>
        )}
      </div>
    </footer>
  )
}
