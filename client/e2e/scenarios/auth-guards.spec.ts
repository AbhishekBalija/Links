import { test, expect } from '@playwright/test'
import { getDatabaseURL, replaceActivationToken } from '../helpers/db'
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
const STUDENT = { email: `e2e-guard-${TS}@test.com`, password: 'E2EPass123', usn: `4MN${String(new Date().getFullYear()).slice(2)}CS${String(TS).slice(-3)}` }
const ZERO_ROLE = { email: `e2e-zerorole-${TS}@test.com`, password: 'E2EPass123', usn: `4MN${String(new Date().getFullYear()).slice(2)}CS${String(TS).slice(-3)}` }

test.describe('Auth Guards', () => {
  let dbURL: string
  let adminToken: string

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    adminToken = await setupAdmin(request, dbURL, admin)
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, [STUDENT.email])
  })

  test('Protected route redirects to /login when logged out', async ({ page }) => {
    await page.goto('/login')
    await page.waitForURL('**/login')

    await page.goto('/')
    await page.waitForURL('**/login')
    await expect(page.locator('h1')).toContainText('Log in')
  })

  test('Logout clears session and redirects to login', async ({ page, request }) => {
    // Onboard the student via real flow
    await submitAccessRequestViaUI(page, {
      full_name: 'Guard Test',
      email: STUDENT.email,
      password: STUDENT.password,
      usn: STUDENT.usn,
      department_code: 'CS',
    })
    await expect(page.locator('h1')).toContainText('Access requested')

    const userId = await getUserIdByEmail(dbURL, STUDENT.email)
    await adminApproveUser(request, adminToken, userId)

    const activationToken = await replaceActivationToken(userId)
    const activationResponse = await request.post('/api/v1/auth/activate', {
      data: { token: activationToken, password: STUDENT.password },
    })
    expect(activationResponse.ok()).toBeTruthy()

    // Login
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await expect(page.locator('h2')).toContainText('Welcome')

    // Logout
    await page.click('button:has-text("Log out")')
    await page.waitForURL('**/login')
    await expect(page.locator('h1')).toContainText('Log in')

    // Protected route blocked
    await page.goto('/')
    await page.waitForURL('**/login')
    await expect(page.locator('h1')).toContainText('Log in')
  })
})

test.describe('Zero-Role User → /account-pending', () => {
  let dbURL: string
  let zeroRoleUserId: string

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    await setupAdmin(request, dbURL, admin)
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, [ZERO_ROLE.email])
  })

  test('Zero-role user lands on /account-pending, not 403 or Dashboard', async ({ page }) => {
    // Onboard user but DO NOT assign a role (zero-role scenario)
    await submitAccessRequestViaUI(page, {
      full_name: 'Zero Role User',
      email: ZERO_ROLE.email,
      password: ZERO_ROLE.password,
      usn: ZERO_ROLE.usn,
      department_code: 'CS',
    })
    await expect(page.locator('h1')).toContainText('Access requested')

    zeroRoleUserId = await getUserIdByEmail(dbURL, ZERO_ROLE.email)
    // Activate directly via DB, NOT via admin endpoint (which also assigns a role).
    // This keeps the user active with zero role assignments (tests ADR-015 /me fix).
    const { Client } = await import('pg')
    const dbClient = new Client({ connectionString: dbURL })
    await dbClient.connect()
    await dbClient.query(`UPDATE users SET status = 'active' WHERE id = $1`, [zeroRoleUserId])
    await dbClient.end()

    // Login — should succeed (no permission gate on /me anymore per ADR-015)
    await loginViaUI(page, ZERO_ROLE.email, ZERO_ROLE.password)

    // Should land on /account-pending, not dashboard
    await page.waitForURL('**/account-pending')
    await expect(page.locator('h1')).toContainText('Account setup incomplete')

    // Logout should work from account-pending
    await page.click('button:has-text("Log out")')
    await page.waitForURL('**/login')
  })
})
