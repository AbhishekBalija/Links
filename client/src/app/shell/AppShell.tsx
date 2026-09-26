import { NavLink, Outlet } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useAuthStore } from '../../features/auth/store'
import { Avatar } from './Avatar'
import { LogoutButton } from './LogoutButton'
import { mainRoleLabel, navFor, type NavItem } from './nav'

// AppShell frames every signed-in screen: a sidebar on desktop and a bottom
// navigation bar on phones, where students do most of their reading.
export function AppShell() {
  const user = useAuthStore((s) => s.user)
  const roles = user?.roles ?? []
  const items = navFor(roles)
  const name = user?.profile.full_name ?? user?.email ?? ''

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
              <SidebarLink item={item} />
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

      <main id="main" className="min-w-0 flex-1 px-4 pt-5 pb-28 lg:px-16 lg:pt-10 lg:pb-12">
        <div className="mx-auto w-full max-w-[1120px]">
          <Outlet />
        </div>
      </main>

      <nav
        aria-label="Main"
        className="fixed inset-x-0 bottom-0 z-40 grid bg-surface pt-2 pb-[max(env(safe-area-inset-bottom),14px)] shadow-[0_-1px_0_var(--color-line)] lg:hidden"
        style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}
      >
        {items.map((item) => (
          <TabLink key={item.to} item={item} />
        ))}
      </nav>
    </div>
  )
}

function SidebarLink({ item }: { item: NavItem }) {
  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-2.5 rounded-lg px-3 py-2.5',
          isActive
            ? 'bg-surface font-semibold text-ink shadow-[0_1px_2px_rgba(27,24,20,0.06)] hover:text-ink'
            : 'text-ink-2 hover:bg-well/60 hover:text-ink',
        )
      }
    >
      {({ isActive }) => (
        <>
          <span aria-hidden="true" className={cn('size-1.5 rounded-full', isActive && 'bg-rust')} />
          {item.label}
        </>
      )}
    </NavLink>
  )
}

function TabLink({ item }: { item: NavItem }) {
  const Icon = item.icon
  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      className={({ isActive }) =>
        cn(
          'flex min-h-12 flex-col items-center justify-center gap-1 text-xs',
          isActive ? 'font-semibold text-rust hover:text-rust' : 'text-ink-2 hover:text-ink',
        )
      }
    >
      <Icon aria-hidden="true" className="size-[22px]" strokeWidth={1.8} />
      {item.label}
    </NavLink>
  )
}
