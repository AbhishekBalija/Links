// Where to go after signing in (#212): opening a shared link to a job, event
// or notice while signed out should land there once signed in, not on Home.
// It is kept for this tab only, and only a page on this site counts, so a
// crafted link can't send someone elsewhere after they sign in.

const key = 'links.returnTo'

// Pages that are part of getting in, or Home itself, are never a target.
const notTargets = new Set(['/', '/login', '/welcome', '/account-pending'])

export function safeReturn(path: string | null | undefined): string | null {
  if (!path || !path.startsWith('/') || path.startsWith('//') || path.includes('\\')) return null
  const pathname = path.split(/[?#]/)[0]
  if (notTargets.has(pathname)) return null
  return path
}

// Storage can be missing or refuse (private windows); then LINKS just goes
// to Home, as before.
export function rememberReturn(path: string) {
  const safe = safeReturn(path)
  if (!safe) return
  try {
    sessionStorage.setItem(key, safe)
  } catch {
    // Nothing to keep it in.
  }
}

export function peekReturn(): string | null {
  try {
    return safeReturn(sessionStorage.getItem(key))
  } catch {
    return null
  }
}

export function forgetReturn() {
  try {
    sessionStorage.removeItem(key)
  } catch {
    // Nothing was kept.
  }
}
