import { useId, useState } from 'react'
import { buttonStyles } from '../../features/announcements/buttons'
import { useAuthStore } from '../../features/auth/store'
import { useModal } from '../../shared/ui/useModal'

// PhoneLogout is log out on phones (#212): at the end of your own Profile,
// named, with the account it signs out of, and a confirm sheet so one tap
// on a shared phone can't sign someone out by accident. Desktop keeps log
// out beside the name in the sidebar.
export function PhoneLogout() {
  const email = useAuthStore((s) => s.user?.email)
  const logout = useAuthStore((s) => s.logout)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)
  const ref = useModal(open)
  const id = useId()

  async function confirm() {
    setBusy(true)
    setFailed(false)
    try {
      await logout()
    } catch {
      setFailed(true)
      setBusy(false)
    }
  }

  return (
    <section aria-label="Your account" className="flex flex-col gap-2.5 rounded-xl border border-line bg-surface px-[18px] py-3.5 lg:hidden">
      {email && (
        <p className="text-[13px] text-ink-3">
          Signed in as <b className="font-semibold break-all text-ink">{email}</b>
        </p>
      )}
      <button type="button" onClick={() => setOpen(true)} className={buttonStyles.secondary + ' w-full'}>
        Log out
      </button>
      <dialog
        ref={ref}
        aria-labelledby={`${id}-h`}
        onCancel={(e) => {
          e.preventDefault()
          setOpen(false)
        }}
        className="m-0 mt-auto w-full max-w-none rounded-t-[18px] bg-paper p-0 text-ink backdrop:bg-ink/45"
      >
        <div className="flex flex-col gap-3 px-5 pt-6 pb-[30px]">
          <h2 id={`${id}-h`} className="font-serif text-2xl font-medium">
            Log out on this phone?
          </h2>
          <p className="mb-1 text-[15px] leading-normal text-ink-2">
            To come back you'll sign in again with a code or Google. You stay signed in on your other devices.
          </p>
          {failed && (
            <p role="alert" className="text-sm font-medium text-danger-ink">
              Could not log out. Try again.
            </p>
          )}
          <button type="button" onClick={confirm} disabled={busy} className={buttonStyles.primary + ' min-h-12 w-full'}>
            {busy ? 'Logging out…' : 'Log out'}
          </button>
          <button type="button" onClick={() => setOpen(false)} className={buttonStyles.secondary + ' w-full'}>
            Cancel
          </button>
        </div>
      </dialog>
    </section>
  )
}
