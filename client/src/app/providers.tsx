import { useEffect, type ReactNode } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { initializeAuth } from '../features/auth/store'

function AuthInit({ children }: { children: ReactNode }) {
  useEffect(() => { initializeAuth() }, [])
  return <>{children}</>
}

export function Providers({ children }: { children: ReactNode }) {
  return (
    <BrowserRouter>
      <AuthInit>
        {children}
      </AuthInit>
    </BrowserRouter>
  )
}
