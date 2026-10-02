import type { ReactNode } from 'react'
import { AlertCircle, Check } from 'lucide-react'
import { cn } from '@/lib/utils'

const tones = {
  danger: 'bg-danger-soft text-danger-ink',
  ok: 'bg-success-soft text-success-ink',
  warn: 'bg-warning-soft text-warning-ink',
  info: 'bg-well text-ink-2',
}

// Notice is the message box above a sign-in form: an error, a warning, or a
// quiet confirmation.
export function Notice({ tone, children }: { tone: keyof typeof tones; children: ReactNode }) {
  const Icon = tone === 'ok' ? Check : AlertCircle
  return (
    <div role={tone === 'danger' ? 'alert' : 'status'} className={cn('flex items-start gap-2.5 rounded-[10px] px-4 py-3.5 text-sm leading-[1.45]', tones[tone])}>
      <Icon aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <p>{children}</p>
    </div>
  )
}
