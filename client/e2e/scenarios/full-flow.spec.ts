import { test, expect } from '@playwright/test'
import { getDatabaseURL } from '../helpers/db'
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
const STUDENT = { email: `e2e-student-${TS}@test.com`, password: 'E2EPass123', usn: `4MN${String(new Date().getFullYear()).slice(2)}CS${String(TS).slice(-3)}` }

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
      usn: STUDENT.usn,
      department_code: 'CS',
    })
    await expect(page.locator('h1')).toContainText('Access requested')
    await expect(page.locator('text=submitted')).toBeVisible()
  })

  test('2. Admin approves user via real endpoint', async ({ request }) => {
    const userId = await getUserIdByEmail(dbURL, STUDENT.email)
    expect(userId).toBeTruthy()
    await adminApproveUser(request, adminToken, userId)
  })

  test('3. Login and see Dashboard', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await expect(page.locator('h2')).toContainText('Welcome')
    await expect(page.locator('strong')).toHaveText('student')
  })

  test('4. Edit Profile — save values', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await page.click('a[href="/profile/edit"]')
    await page.waitForURL('**/profile/edit')

    await page.fill('#headline', 'Computer Science Student')
    await page.fill('#bio', 'A passionate developer building cool things.')
    await page.fill('#linkedin', 'https://linkedin.com/in/e2e-test')
    await page.fill('#github', 'https://github.com/e2e-test')
    await page.fill('#portfolio', 'https://e2e-test.dev')
    await page.click('button:has-text("Save")')

    await page.waitForURL('**/')
    await expect(page.locator('h2')).toContainText('Welcome')
  })

  test('5. Profile edits persist after navigation', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await page.click('a[href="/profile/edit"]')
    await page.waitForURL('**/profile/edit')

    await expect(page.locator('#headline')).toHaveValue('Computer Science Student')
    await expect(page.locator('#bio')).toHaveValue('A passionate developer building cool things.')
    await expect(page.locator('#linkedin')).toHaveValue('https://linkedin.com/in/e2e-test')
    await expect(page.locator('#github')).toHaveValue('https://github.com/e2e-test')
    await expect(page.locator('#portfolio')).toHaveValue('https://e2e-test.dev')
  })

  test('6. Session persists on page refresh', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await expect(page.locator('h2')).toContainText('Welcome')

    await page.reload()
    await page.waitForLoadState('networkidle')

    await expect(page.locator('h2')).toContainText('Welcome')
    await expect(page.locator('button:has-text("Log out")')).toBeVisible()
  })

  test('7. Profile edits survive page refresh', async ({ page }) => {
    await loginViaUI(page, STUDENT.email, STUDENT.password)
    await page.click('a[href="/profile/edit"]')
    await page.waitForURL('**/profile/edit')

    await expect(page.locator('#headline')).toHaveValue('Computer Science Student')
  })
})
