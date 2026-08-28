import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '../../../components/ui/button'
import { useAuthStore } from '../../auth/store'

export default function Dashboard() {
	const user = useAuthStore((s) => s.user)
	const logout = useAuthStore((s) => s.logout)
	const [logoutError, setLogoutError] = useState('')

	const handleLogout = async () => {
		setLogoutError('')
		try {
			await logout()
		} catch {
			setLogoutError('Could not log out. Your session is still active; please try again.')
		}
	}

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="flex items-center justify-between border-b border-border bg-white px-6 py-3">
        <span className="text-lg font-bold text-foreground">LINKS</span>
        <div className="flex items-center gap-4">
          <span className="text-sm text-muted-foreground">{user?.profile.full_name ?? user?.email}</span>
			<Button variant="outline" size="sm" onClick={handleLogout}>Log out</Button>
		</div>
	</header>
	{logoutError && <p role="alert" className="border-b border-border px-6 py-2 text-sm text-destructive">{logoutError}</p>}
      <main className="mx-auto flex-1 p-8 w-full max-w-4xl">
        <h2 className="text-xl font-semibold text-foreground mb-2">
          Welcome, {user?.profile.full_name ?? 'User'}
        </h2>
        <p className="text-sm text-muted-foreground mb-6">
          You're logged in as <strong className="text-foreground">{user?.roles.join(', ') || 'no role'}</strong>.
        </p>
        <Link to="/profile/edit" className="inline-block">
          <Button variant="outline" size="sm">Edit Profile</Button>
        </Link>
      </main>
    </div>
  )
}
