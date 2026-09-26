import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

// Skeleton is a placeholder bar shown while content loads.
export function Skeleton({ className }: { className?: string }) {
  return <span aria-hidden="true" className={cn('block rounded bg-skeleton', className)} />
}

// LoadingStatus tells screen readers what the skeletons mean.
export function LoadingStatus({ label }: { label: string }) {
  return (
    <p role="status" className="sr-only">
      {label}
    </p>
  )
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="flex flex-col items-start gap-2 rounded-xl border border-line bg-surface px-6 py-8">
      <h2 className="font-serif text-xl font-medium text-ink">{title}</h2>
      {children && <div className="text-sm text-ink-2">{children}</div>}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div role="alert" className="flex flex-col items-start gap-3 rounded-xl bg-danger-soft px-6 py-5">
      <p className="text-sm font-semibold text-danger-ink">{message}</p>
      <button
        type="button"
        onClick={onRetry}
        className="min-h-11 rounded-lg bg-ink px-4 text-sm font-semibold text-paper hover:bg-ink-2"
      >
        Try again
      </button>
    </div>
  )
}
