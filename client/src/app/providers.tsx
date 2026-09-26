import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect, useState, type ReactNode } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { initializeAuth, useAuthStore } from '../features/auth/store'
import { ApiRequestError } from '../shared/api/types'

function AuthInit({ children }: { children: ReactNode }) {
  useEffect(() => { initializeAuth() }, [])
  return <>{children}</>
}

// A 4xx answer won't change on retry (a notice that isn't yours stays that
// way), so only network and server errors are retried.
function shouldRetry(failureCount: number, error: unknown) {
  if (error instanceof ApiRequestError && error.status < 500) return false
  return failureCount < 2
}

// ClearOnUserChange drops cached data when someone else signs in on the same
// tab, so one person's notices never flash on another person's screen.
function ClearOnUserChange({ client }: { client: QueryClient }) {
  const userId = useAuthStore((s) => s.user?.id ?? null)
  useEffect(() => {
    return () => client.clear()
  }, [client, userId])
  return null
}

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: shouldRetry, staleTime: 30_000 } } }),
  )
  return (
    <QueryClientProvider client={queryClient}>
      <ClearOnUserChange client={queryClient} />
      <BrowserRouter>
        <AuthInit>
          {children}
        </AuthInit>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
