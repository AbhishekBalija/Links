import { test, expect, type Page } from '@playwright/test'
import { getDatabaseURL } from '../helpers/db'
import {
  bootstrapAdmin,
  setupAdmin,
  loginViaUI,
  loginViaAPI,
  signInWithCode,
  freeUSN,
  getUserIdByEmail,
  adminApproveUser,
  cleanupTestUsers,
  expectHome,
} from '../helpers/auth'

// openEditProfile goes the way a member would: Profile in the sidebar, then
// Edit profile on their profile page.
async function openEditProfile(page: Page) {
  await page.getByRole('navigation', { name: 'Main' }).first().getByRole('link', { name: 'Profile' }).click()
  await page.waitForURL('**/profile')
  await page.getByRole('link', { name: 'Edit profile' }).click()
  await page.waitForURL('**/profile/edit')
}

const TS = Date.now()
const STUDENT = { email: `e2e-student-${TS}@test.com` }

test.describe.serial('Full E2E Flow: Student (real onboarding)', () => {
  let dbURL: string
  let adminEmail: string

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    await setupAdmin(request, dbURL, admin)
    adminEmail = admin.email
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, [STUDENT.email])
  })

  test('1. Someone on no class list signs in with an email code and sends a request', async ({ page }) => {
    const usn = await freeUSN('CS', 2024)
    await signInWithCode(page, STUDENT.email)
    await expect(page.getByRole('heading', { name: "You're not on a class list yet" })).toBeVisible()
    await page.getByLabel('Your name').fill('E2E Student')
    await page.getByLabel('USN').fill(usn)
    await page.getByRole('button', { name: 'Send request to the CS HOD' }).click()
    await expect(page.getByRole('heading', { name: 'Your request is with the CS HOD' })).toBeVisible()
  })

  test('2. Admin approves the request via the real endpoint', async ({ request }) => {
    const userId = await getUserIdByEmail(dbURL, STUDENT.email)
    expect(userId).toBeTruthy()
    // Access tokens last seconds in the e2e suite, so ask for a fresh one.
    await adminApproveUser(request, await loginViaAPI(request, adminEmail), userId)
  })

  test('3. The approved student signs in, confirms who they are and sees Home', async ({ page }) => {
    await signInWithCode(page, STUDENT.email)
    await page.waitForURL('**/welcome')
    await expect(page.getByRole('heading', { name: 'Welcome to Links, E2E' })).toBeVisible()
    await page.getByRole('button', { name: "Yes, that's me" }).click()
    await expectHome(page)
    await expect(page.getByRole('navigation', { name: 'Main' }).getByText('Student', { exact: true })).toBeVisible()
  })

  test('5. Edit Profile: save values', async ({ page }) => {
    await loginViaUI(page, STUDENT.email)
    await openEditProfile(page)

    await page.fill('#headline', 'Computer Science Student')
    await page.fill('#bio', 'A passionate developer building cool things.')
    await page.fill('#linkedin', 'https://linkedin.com/in/e2e-test')
	await page.fill('#github', 'https://github.com/e2e-test')
	await page.fill('#portfolio', 'https://e2e-test.dev')
	await page.check('input[type="checkbox"]')
	await page.click('button:has-text("Save")')

    // Saving returns to the profile page, which shows the new headline.
    await page.waitForURL('**/profile')
    await expect(page.getByText('Computer Science Student')).toBeVisible()
  })

  test('6. Profile edits persist after navigation', async ({ page }) => {
    await loginViaUI(page, STUDENT.email)
    await openEditProfile(page)

    await expect(page.locator('#headline')).toHaveValue('Computer Science Student')
    await expect(page.locator('#bio')).toHaveValue('A passionate developer building cool things.')
    await expect(page.locator('#linkedin')).toHaveValue('https://linkedin.com/in/e2e-test')
    await expect(page.locator('#github')).toHaveValue('https://github.com/e2e-test')
	await expect(page.locator('#portfolio')).toHaveValue('https://e2e-test.dev')
	await expect(page.locator('input[type="checkbox"]').first()).toBeChecked()
  })

  test('7. Session persists on page refresh', async ({ page, context }) => {
    await loginViaUI(page, STUDENT.email)
    await expectHome(page)

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

    await expectHome(page)
    await expect(page.getByRole('button', { name: 'Log out' })).toBeVisible()
  })

	test('8. Profile edits survive refresh and optional fields can be cleared', async ({ page }) => {
		await loginViaUI(page, STUDENT.email)
		await openEditProfile(page)

		await expect(page.locator('#headline')).toHaveValue('Computer Science Student')

		await page.fill('#headline', '')
		await page.fill('#bio', '')
		await page.fill('#linkedin', '')
		await page.fill('#github', '')
		await page.fill('#portfolio', '')
		await page.click('button:has-text("Save")')
		await page.waitForURL('**/profile')
		await openEditProfile(page)
		await expect(page.locator('#headline')).toHaveValue('')
		await expect(page.locator('#linkedin')).toHaveValue('')
	})
})
