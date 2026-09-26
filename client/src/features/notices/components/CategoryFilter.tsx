import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { categories, type Category } from '../types'

// The filter is a row of links (?category=...), so the choice survives a
// reload, works with the back button and needs no extra keyboard handling.
export function CategoryFilter({ current }: { current: Category | null }) {
  const options = [{ value: null, label: 'All', short: 'All' }, ...categories]
  return (
    <nav aria-label="Filter by category" className="flex gap-1 rounded-[10px] bg-well p-1 lg:self-start">
      {options.map((option) => {
        const active = option.value === current
        return (
          <Link
            key={option.label}
            to={option.value ? `?category=${option.value}` : '.'}
            aria-current={active ? 'true' : undefined}
            className={cn(
              'flex min-h-9 flex-1 items-center justify-center rounded-[7px] px-2 text-[13px] lg:flex-none lg:px-[18px] lg:text-sm',
              active
                ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)] hover:text-ink'
                : 'text-ink-2 hover:text-ink',
            )}
          >
            <span className="lg:hidden">{option.short}</span>
            <span className="hidden lg:inline">{option.label}</span>
          </Link>
        )
      })}
    </nav>
  )
}
