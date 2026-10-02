import { useState } from 'react'
import { useAuthStore } from '../store'
import { SignInHeading, SignInLayout } from '../components/SignInLayout'
import { Notice } from '../components/Notice'

// AccountPending is for someone signed in whose account has no role yet,
// such as one whose last role just ended.
export default function AccountPending() {
  const logout = useAuthStore((s) => s.logout)
  const [failed, setFailed] = useState(false)

  return (
    <SignInLayout>
      <SignInHeading label="No role yet">Your account has no role yet</SignInHeading>
      <p className="text-[15px] leading-normal text-ink-2">You're signed in, but nothing is set up for you to do here yet.</p>
      <Notice tone="info">Ask your HOD or an admin to give you a role. Then sign in again.</Notice>
      {failed && <Notice tone="danger">Could not sign out. Try again.</Notice>}
      <button
        type="button"
        onClick={() => {
          setFailed(false)
          logout().catch(() => setFailed(true))
        }}
        className="min-h-11 self-center text-sm font-semibold text-rust"
      >
        Sign out
      </button>
    </SignInLayout>
  )
}
