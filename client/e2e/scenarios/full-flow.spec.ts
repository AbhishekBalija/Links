import { test, expect } from '@playwright/test'
import { getDatabaseURL, getSchemaClient, replaceActivationToken } from '../helpers/db'
import {
  bootstrapAdmin,
  setupAdmin,
  loginViaUI,
  submitAccessRequestViaUI,
  getUserIdByEmail,
  adminApproveUser,
  cleanupTestUsers,
} from '../helpers/auth'

const TS = Date.now()
const STUDENT = { email: `e2e-student-${TS}@test.com`, password: 'E2EPass123', phone: '+91 98765 43210', usn: `4MN${String(new Date().getFullYear()).slice(2)}CS${String(TS).slice(-3)}` }

test.describe.serial('Full E2E Flow: Student (real onboarding)', () => {
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

  test('1. Submit access request via real UI form', async ({ page }) => {
    await submitAccessRequestViaUI(page, {
      full_name: 'E2E Student',
      email: STUDENT.email,
		password: STUDENT.password,
		phone: STUDENT.phone,
      usn: STUDENT.usn,
      department_code: 'CS',
    })
	await expect(page.locator('h1')).toContainText('Access requested')
	await expect(page.locator('text=submitted')).toBeVisible()

	const dbClient = await getSchemaClient()
	try {
		const result = await dbClient.query('SELECT phone FROM users WHERE email = $1', [STUDENT.email])
		expect(result.rows[0]?.phone).toBe(STUDENT.phone)
	} finally {
		await dbClient.end()
	}
  })

  test('2. Admin approves user via real endpoint', async ({ request }) => {
    const userId = await getUserIdByEmail(dbURL, STUDENT.email)
    expect(userId).toBeTruthy()
    await adminApproveUser(request, adminToken, userId)
  })

  test('3. Approved user must activate before logging in', async ({ request }) => {
    const loginResponse = await request.post('/api/v1/auth/login', {
      data: { email: STUDENT.email, password: STUDENT.password },
    })
    expect(loginResponse.status()).toBe(401)
  })

	test('4. Activate account through the email-link UI, then login and see Dashboard', async ({ page }) => {
		const userId = await getUserIdByEmail(dbURL, STUDENT.email)
		const activationToken = await replaceActivationToken(userId)
		await page.goto(`/activate?token=${activationToken}`)
		await page.fill('#password', STUDENT.password)
		await page.fill('#password-confirmation', STUDENT.password)
		await page.click('button:has-text("Activate account")')
		await expect(page.locator('h1')).toContainText('Account activated')

    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await expect(page.locator('h2')).toContainText('Welcome')
    await expect(page.locator('strong')).toHaveText('student')
  })

  test('5. Edit Profile — save values', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await page.click('a[href="/profile/edit"]')
    await page.waitForURL('**/profile/edit')

    await page.fill('#headline', 'Computer Science Student')
    await page.fill('#bio', 'A passionate developer building cool things.')
    await page.fill('#linkedin', 'https://linkedin.com/in/e2e-test')
	await page.fill('#github', 'https://github.com/e2e-test')
	await page.fill('#portfolio', 'https://e2e-test.dev')
	await page.check('input[type="checkbox"]')
	await page.click('button:has-text("Save")')

    await page.waitForURL('**/')
    await expect(page.locator('h2')).toContainText('Welcome')
  })

  test('6. Profile edits persist after navigation', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await page.click('a[href="/profile/edit"]')
    await page.waitForURL('**/profile/edit')

    await expect(page.locator('#headline')).toHaveValue('Computer Science Student')
    await expect(page.locator('#bio')).toHaveValue('A passionate developer building cool things.')
    await expect(page.locator('#linkedin')).toHaveValue('https://linkedin.com/in/e2e-test')
    await expect(page.locator('#github')).toHaveValue('https://github.com/e2e-test')
	await expect(page.locator('#portfolio')).toHaveValue('https://e2e-test.dev')
	await expect(page.locator('input[type="checkbox"]').first()).toBeChecked()
  })

  test('7. Session persists on page refresh', async ({ page, context }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await expect(page.locator('h2')).toContainText('Welcome')

    const refreshCookie = (await context.cookies()).find((cookie) => cookie.name === 'refresh_token')
    expect(refreshCookie).toBeDefined()

    const refreshResponse = page.waitForResponse((response) => {
      const url = new URL(response.url())
      return url.pathname === '/api/v1/auth/refresh' && response.request().method() === 'POST'
    })

    await page.reload()
    await page.waitForLoadState('networkidle')

    const response = await refreshResponse
    expect(response.status()).toBe(200)
    await expect(response.json()).resolves.toMatchObject({
      data: { access_token: expect.any(String) },
    })

    await expect(page.locator('h2')).toContainText('Welcome')
    await expect(page.locator('button:has-text("Log out")')).toBeVisible()
  })

	test('8. Profile edits survive refresh and optional fields can be cleared', async ({ page }) => {
		await loginViaUI(page, STUDENT.email, STUDENT.password)
		await page.click('a[href="/profile/edit"]')
		await page.waitForURL('**/profile/edit')

		await expect(page.locator('#headline')).toHaveValue('Computer Science Student')

		await page.fill('#headline', '')
		await page.fill('#bio', '')
		await page.fill('#linkedin', '')
		await page.fill('#github', '')
		await page.fill('#portfolio', '')
		await page.click('button:has-text("Save")')
		await page.waitForURL('**/')
		await page.click('a[href="/profile/edit"]')
		await page.waitForURL('**/profile/edit')
		await expect(page.locator('#headline')).toHaveValue('')
		await expect(page.locator('#linkedin')).toHaveValue('')
	})
})
