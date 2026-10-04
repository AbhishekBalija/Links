import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'

export type WorkspaceTab = { label: string; to: string; count?: number | string }

// WorkspaceTabs switches between the parts of a workspace: Posts and Access
// requests in an HOD's Approval queue, the admin tools in Admin.
export function WorkspaceTabs({ tabs, active, label }: { tabs: WorkspaceTab[]; active: string; label: string }) {
  return (
    <nav aria-label={label} className="flex gap-1 self-stretch overflow-x-auto rounded-[10px] bg-well p-1 lg:self-start">
      {tabs.map((tab) => {
        const on = tab.label === active
        return (
          <Link
            key={tab.label}
            to={tab.to}
            replace
            aria-current={on ? 'page' : undefined}
            className={cn(
              'flex min-h-10 flex-1 items-center justify-center gap-1.5 rounded-[7px] px-3 text-sm whitespace-nowrap lg:flex-none lg:px-4',
              on ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.08)]' : 'text-ink-2 hover:text-ink',
            )}
          >
            {tab.label}
            {tab.count !== undefined && <span className={cn('font-mono text-xs', on ? 'text-rust' : 'text-ink-3')}>{tab.count}</span>}
          </Link>
        )
      })}
    </nav>
  )
}
