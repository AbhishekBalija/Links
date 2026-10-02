import { useEffect, useRef, useState } from 'react'
import { googleNonce } from '../api'
import { googleClientId } from '../google'

// The parts of Google Identity Services this button uses.
type GoogleIdentity = {
  accounts: {
    id: {
      initialize: (config: { client_id: string; nonce: string; callback: (response: { credential: string }) => void; ux_mode: 'popup' }) => void
      renderButton: (parent: HTMLElement, options: { theme: 'filled_black'; size: 'large'; text: 'continue_with'; shape: 'rectangular'; width: number }) => void
    }
  }
}

declare global {
  interface Window {
    google?: GoogleIdentity
  }
}

let scriptLoad: Promise<GoogleIdentity> | null = null

function loadGoogle(): Promise<GoogleIdentity> {
  if (window.google) return Promise.resolve(window.google)
  scriptLoad ??= new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    script.async = true
    script.onload = () => (window.google ? resolve(window.google) : reject(new Error('Google sign-in did not load')))
    script.onerror = () => {
      scriptLoad = null
      reject(new Error('Google sign-in did not load'))
    }
    document.head.appendChild(script)
  })
  return scriptLoad
}

// GoogleButton renders Google's own "Continue with Google" button, which the
// sign-in popup needs. Each mount asks the server for a fresh nonce, so remount
// it (change its key) after a failed try.
export function GoogleButton({ onCredential, onError }: { onCredential: (credential: string) => void; onError: () => void }) {
  const ref = useRef<HTMLDivElement>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    Promise.all([loadGoogle(), googleNonce()])
      .then(([google, { nonce }]) => {
        const parent = ref.current
        if (cancelled || !parent) return
        google.accounts.id.initialize({ client_id: googleClientId, nonce, ux_mode: 'popup', callback: (response) => onCredential(response.credential) })
        google.accounts.id.renderButton(parent, { theme: 'filled_black', size: 'large', text: 'continue_with', shape: 'rectangular', width: Math.min(parent.clientWidth, 400) })
        setReady(true)
      })
      .catch(() => {
        if (!cancelled) onError()
      })
    return () => {
      cancelled = true
    }
  }, [onCredential, onError])

  return (
    <div className="relative min-h-11">
      {!ready && <div aria-hidden="true" className="absolute inset-0 rounded-lg bg-ink/90" />}
      <div ref={ref} className="flex justify-center [&_iframe]:!w-full" />
    </div>
  )
}
