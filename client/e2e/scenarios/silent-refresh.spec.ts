import { test, expect } from '@playwright/test'
import { getDatabaseURL } from '../helpers/db'

declare global {
  interface Window {
    __apiRequest: <T>(path: string) => Promise<T>
  }
}
import {
  bootstrapAdmin,
  setupAdmin,
  loginViaUI,
  cleanupTestUsers,
  getUserIdByEmail,
  adminApproveUser,
  submitAccessRequestViaUI,
} from '../helpers/auth'

const TS = Date.now()
const USER = { email: `e2e-refresh-${TS}@test.com`, password: 'E2EPass123', usn: `4MN${String(new Date().getFullYear()).slice(2)}CS${String(TS).slice(-3)}` }

test.describe('Silent Token Refresh', () => {
  let dbURL: string
  let adminToken: string

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    adminToken = await setupAdmin(request, dbURL, admin)
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, [USER.email])
  })

  test('Session continues after token expiry via silent refresh', async ({ page, context, request }) => {
    // 1. Onboard user via real flow
    await submitAccessRequestViaUI(page, {
      full_name: 'Refresh Test',
      email: USER.email,
      password: USER.password,
      usn: USER.usn,
      department_code: 'CS',
    })
    await expect(page.locator('h1')).toContainText('Access requested')

    const userId = await getUserIdByEmail(dbURL, USER.email)
    await adminApproveUser(request, adminToken, userId)

    // 2. Log in via UI — this sets refresh_token cookie in browser
    await loginViaUI(page, USER.email, USER.password)
    await expect(page.locator('h2')).toContainText('Welcome')

    // 3. Capture current refresh cookie value
    const cookiesBefore = await context.cookies()
    const refreshBefore = cookiesBefore.find((c) => c.name === 'refresh_token')
    expect(refreshBefore).toBeDefined()
    console.log(`  Refresh cookie (before): ${refreshBefore!.value.slice(0, 20)}...`)

    // 4. Wait for access token expiry (TTL=10s, wait 14s for margin)
    console.log('  Waiting 14s for token expiry...')
    await page.waitForTimeout(14000)

    // 5. Trigger an API call via the app's apiRequest (exposed on window in dev).
    //    The interceptor catches the 401, calls /auth/refresh using the cookie,
    //    gets a new token, and retries the original request.
    const result = await page.evaluate(async () => {
      const apiRequest = window.__apiRequest
      if (!apiRequest) return { ok: false, error: '__apiRequest not exposed' }
      try {
        const data = await apiRequest<{ email: string }>('/api/v1/me')
        return { ok: true, email: data.email }
      } catch (err) {
        return { ok: false, error: err instanceof Error ? err.message : String(err) }
      }
    })
    expect(result.ok).toBe(true)
    expect(result.email).toBe(USER.email)

    // 6. Refresh token should have been rotated server-side
    const cookiesAfter = await context.cookies()
    const refreshAfter = cookiesAfter.find((c) => c.name === 'refresh_token')
    expect(refreshAfter).toBeDefined()
    const rotated = refreshBefore!.value !== refreshAfter!.value
    console.log(`  Refresh cookie (after):  ${refreshAfter!.value.slice(0, 20)}...`)
    console.log(`  Token rotated: ${rotated}`)
    expect(rotated).toBe(true)

    // 7. Session still works (same-page SPA navigation, not full reload)
    await expect(page.locator('h2')).toContainText('Welcome')
  })
})
