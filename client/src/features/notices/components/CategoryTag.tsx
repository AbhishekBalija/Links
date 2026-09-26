import { cn } from '@/lib/utils'
import { categories, type Category } from '../types'

const tagStyles: Record<Category, string> = {
  official: 'bg-tag-official text-tag-official-ink',
  department: 'bg-tag-department text-tag-department-ink',
  placement: 'bg-tag-placement text-tag-placement-ink',
}

export function CategoryTag({ category, className }: { category: Category; className?: string }) {
  const label = categories.find((c) => c.value === category)?.label ?? category
  return (
    <span className={cn('inline-block rounded px-1.5 py-0.5 text-[11px] font-semibold leading-4', tagStyles[category], className)}>
      {label}
    </span>
  )
}
