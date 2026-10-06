import { useState } from 'react'
import { cn } from '@/lib/utils'
import { cleanCode } from '../signIn'

// CodeBoxes is one real input drawn as six boxes, so phones can fill it from
// the email and screen readers hear a single field.
export function CodeBoxes({ value, onChange, invalid, describedBy }: { value: string; onChange: (code: string) => void; invalid: boolean; describedBy?: string }) {
  const [focused, setFocused] = useState(false)
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor="code" className="text-sm font-semibold">
        6-digit code
      </label>
      <div className="relative flex gap-2">
        {Array.from({ length: 6 }, (_, i) => (
          <span
            key={i}
            aria-hidden="true"
            className={cn(
              'flex h-[58px] w-12 items-center justify-center rounded-lg border bg-surface font-mono text-[26px] font-medium',
              invalid ? 'border-danger border-[1.5px]' : 'border-line',
              focused && !invalid && i === Math.min(value.length, 5) && 'ring-2 ring-rust',
            )}
          >
            {value[i] ?? ''}
          </span>
        ))}
        <input
          id="code"
          inputMode="numeric"
          autoComplete="one-time-code"
          value={value}
          aria-invalid={invalid || undefined}
          aria-describedby={describedBy}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onChange={(e) => onChange(cleanCode(e.target.value))}
          className="absolute inset-0 h-full w-full cursor-text opacity-0"
        />
      </div>
    </div>
  )
}
