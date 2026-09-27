import { Suspense } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { PageLoading } from '../../shared/ui/states'
import { cn } from '@/lib/utils'
import { useAuthStore } from '../../features/auth/store'
import { Avatar } from './Avatar'
import { LogoutButton } from './LogoutButton'
import { useDashboard } from '../../features/home/api'
import { canApprove, mainRoleLabel, navFor, tabsFor, type NavItem } from './nav'

// AppShell frames every signed-in screen: a sidebar on desktop and a bottom
// navigation bar on phones, where students do most of their reading.
export function AppShell() {
  const user = useAuthStore((s) => s.user)
  const roles = user?.roles ?? []
  const items = navFor(roles)
  const tabs = tabsFor(roles)
  const name = user?.profile.full_name ?? user?.email ?? ''
  // The composer, an opened announcement and an event have their own action
  // bar at the bottom on phones, so the tab bar steps aside there.
  const focused = /^\/(mine|approvals|events)\/.+/.test(useLocation().pathname)
  // Approvers see how many announcements are waiting for them.
  const approver = canApprove(roles)
  const dashboard = useDashboard(approver)
  const waiting = approver ? dashboard.data?.approvals?.pending_count : undefined
  const badges: Record<string, number | undefined> = { '/approvals': waiting || undefined }

  return (
    <div className="min-h-dvh bg-paper text-ink lg:flex">
      <a
        href="#main"
        className="sr-only z-50 rounded-lg bg-ink px-4 py-2 text-paper focus:not-sr-only focus:fixed focus:top-3 focus:left-3 hover:text-paper"
      >
        Skip to content
      </a>

      <nav aria-label="Main" className="sticky top-0 hidden h-dvh w-64 shrink-0 flex-col gap-8 bg-rail px-5 py-8 lg:flex">
        <div className="flex flex-col gap-1 px-3">
          <span className="font-serif text-[30px] leading-none font-semibold tracking-[-0.4px]">Links</span>
          <span className="font-mono text-[11px] uppercase tracking-[1.2px] text-ink-3">Campus hub</span>
        </div>
        <ul className="flex flex-col gap-1 text-[15px]">
          {items.map((item) => (
            <li key={item.to}>
              <SidebarLink item={item} badge={badges[item.to]} />
            </li>
          ))}
        </ul>
        <div className="mt-auto flex items-center gap-2.5 pl-3">
          <Avatar name={name} />
          <div className="flex min-w-0 flex-1 flex-col text-[13px]">
            <span className="truncate font-semibold">{name}</span>
            <span className="truncate text-ink-3">{mainRoleLabel(roles)}</span>
          </div>
          <LogoutButton iconOnly />
        </div>
      </nav>

      <main id="main" className={cn('min-w-0 flex-1 px-4 pt-5 lg:px-16 lg:pt-10 lg:pb-12', focused ? 'pb-8' : 'pb-28')}>
        <div className="mx-auto w-full max-w-[1120px]">
          <Suspense fallback={<PageLoading />}>
            <Outlet />
          </Suspense>
        </div>
      </main>

      <nav
        aria-label="Main"
        hidden={focused}
        className="fixed inset-x-0 bottom-0 z-40 grid bg-surface pt-2 pb-[max(env(safe-area-inset-bottom),14px)] shadow-[0_-1px_0_var(--color-line)] lg:hidden"
        style={{ gridTemplateColumns: `repeat(${tabs.length}, minmax(0, 1fr))` }}
      >
        {tabs.map((item) => (
          <TabLink key={item.to} item={item} badge={badges[item.to]} />
        ))}
      </nav>
    </div>
  )
}

// useAlsoActive is true on the other paths an item owns, such as a
// Department page under People.
function useAlsoActive(item: NavItem) {
  const path = useLocation().pathname
  return item.also?.some((prefix) => path.startsWith(prefix)) ?? false
}

function SidebarLink({ item, badge }: { item: NavItem; badge?: number }) {
  const also = useAlsoActive(item)
  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      aria-current={also ? 'page' : undefined}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-2.5 rounded-lg px-3 py-2.5',
          isActive || also
            ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.06)] hover:text-ink'
            : 'text-ink-2 hover:bg-well/60 hover:text-ink',
        )
      }
    >
      {({ isActive }) => (
        <>
          <span aria-hidden="true" className={cn('size-1.5 rounded-full', (isActive || also) && 'bg-rust')} />
          <span className="flex-1">{item.label}</span>
          {badge !== undefined && <Count n={badge} />}
        </>
      )}
    </NavLink>
  )
}

function TabLink({ item, badge }: { item: NavItem; badge?: number }) {
  const Icon = item.icon
  const also = useAlsoActive(item)
  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      aria-current={also ? 'page' : undefined}
      className={({ isActive }) =>
        cn(
          'flex min-h-12 flex-col items-center justify-center gap-1 text-xs',
          isActive || also ? 'font-semibold text-rust hover:text-rust' : 'text-ink-2 hover:text-ink',
        )
      }
    >
      <span className="relative">
        <Icon aria-hidden="true" className="size-[22px]" strokeWidth={1.8} />
        {badge !== undefined && (
          <span className="absolute -top-2 left-[21px]">
            <Count n={badge} small />
          </span>
        )}
      </span>
      {item.short ?? item.label}
    </NavLink>
  )
}

function Count({ n, small }: { n: number; small?: boolean }) {
  return (
    <span className={cn('rounded-full bg-rust font-mono font-medium text-surface', small ? 'px-[5px] text-[10px] leading-4' : 'px-[7px] py-px text-[11px]')}>
      {n}
      <span className="sr-only"> waiting</span>
    </span>
  )
}
