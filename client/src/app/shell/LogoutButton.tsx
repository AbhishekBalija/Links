import { LogOut } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import { useAuthStore } from '../../features/auth/store'

// LogoutButton signs the user out. If the server call fails the session is
// still active, so it says so instead of pretending the user is signed out.
export function LogoutButton({ iconOnly, className }: { iconOnly?: boolean; className?: string }) {
  const logout = useAuthStore((s) => s.logout)
  const [failed, setFailed] = useState(false)

  async function handleClick() {
    setFailed(false)
    try {
      await logout()
    } catch {
      setFailed(true)
    }
  }

  return (
    <span className="flex flex-col items-end gap-1">
      <button
        type="button"
        onClick={handleClick}
        aria-label={iconOnly ? 'Log out' : undefined}
        title={iconOnly ? 'Log out' : undefined}
        className={cn(
          'inline-flex min-h-11 items-center gap-2 rounded-lg text-sm font-medium text-ink-2 hover:bg-well hover:text-ink',
          iconOnly ? 'min-w-11 justify-center' : 'px-3',
          className,
        )}
      >
        <LogOut aria-hidden="true" className="size-[18px]" />
        {!iconOnly && 'Log out'}
      </button>
      {failed && (
        <span role="alert" className="text-xs font-medium text-danger">
          Could not log out. Try again.
        </span>
      )}
    </span>
  )
}
