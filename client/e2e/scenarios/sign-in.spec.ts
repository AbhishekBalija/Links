import { test, expect, type APIRequestContext } from '@playwright/test'
import { getDatabaseURL } from '../helpers/db'
import {
  adminApproveUser,
  bootstrapAdmin,
  cleanupTestUsers,
  expectHome,
  freeUSN,
  getUserIdByEmail,
  importStudents,
  lastCode,
  loginViaAPI,
  setupAdmin,
  signInWithCode,
  typeCode,
} from '../helpers/auth'

// Getting in (spec #129): an email code against the class list. Google can't
// run in CI; it is checked by hand with the test user before a release.
const TS = Date.now()
const LISTED = `e2e-listed-${TS}@test.com`
const UNLISTED = `e2e-unlisted-${TS}@test.com`
const REPORTER = `e2e-reporter-${TS}@test.com`
const BATCH = 2023

test.describe('Sign in with an email code', () => {
  let dbURL: string
  let adminEmail: string
  // Access tokens last seconds in the e2e suite, so each step asks for one.
  const adminToken = (request: APIRequestContext) => loginViaAPI(request, adminEmail)

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    await setupAdmin(request, dbURL, admin)
    adminEmail = admin.email
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, [LISTED, UNLISTED, REPORTER])
  })

  test('a student on the class list signs in, confirms who they are and lands on Home', async ({ page, request }) => {
    const usn = await freeUSN('CS', BATCH)
    await importStudents(request, await adminToken(request), [{ email: LISTED, fullName: 'Listed Student', usn }])

    await page.goto('/login')
    await page.getByLabel('Email').fill(LISTED)
    await page.getByRole('button', { name: 'Email me a code' }).click()
    await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()

    // A wrong code is refused, and the right one still works after it.
    const code = await lastCode(request, LISTED)
    await typeCode(page, code === '000000' ? '000001' : '000000')
    await expect(page.getByRole('alert')).toContainText("That code didn't work")
    await typeCode(page, code)

    await page.waitForURL('**/welcome')
    await expect(page.getByRole('heading', { name: 'Welcome to Links, Listed' })).toBeVisible()
    await expect(page.getByText(usn)).toBeVisible()
    await page.getByRole('button', { name: "Yes, that's me" }).click()
    await expectHome(page)

    // Signing out says so on the sign-in screen.
    await page.getByRole('button', { name: 'Log out' }).first().click()
    await page.waitForURL('**/login')
    await expect(page.getByText("You're signed out on this device.")).toBeVisible()
  })

  test('someone on no list sends a request, and once approved signs in', async ({ page, request }) => {
    const usn = await freeUSN('CS', BATCH)
    await signInWithCode(page, UNLISTED)

    await expect(page.getByRole('heading', { name: "You're not on a class list yet" })).toBeVisible()
    await expect(page.getByText('Proved with an email code')).toBeVisible()
    await page.getByLabel('Your name').fill('Unlisted Student')
    await page.getByLabel('USN').fill(usn.toLowerCase())
    await expect(page.getByText('Computer Science and Engineering, 2023')).toBeVisible()
    await page.getByRole('button', { name: 'Send request to the CS HOD' }).click()
    await expect(page.getByRole('heading', { name: 'Your request is with the CS HOD' })).toBeVisible()

    // Signing in before anyone decides says the request is still waiting.
    await signInWithCode(page, UNLISTED)
    await expect(page.getByRole('heading', { name: 'Your request is still waiting' })).toBeVisible()

    await adminApproveUser(request, await adminToken(request), await getUserIdByEmail(dbURL, UNLISTED))
    await signInWithCode(page, UNLISTED)
    await page.waitForURL('**/welcome')
    await page.getByRole('button', { name: "Yes, that's me" }).click()
    await expectHome(page)
  })

  test('"Not you?" on a first sign-in signs out and blocks the row until it is checked', async ({ page, request }) => {
    const usn = await freeUSN('CS', BATCH)
    await importStudents(request, await adminToken(request), [{ email: REPORTER, fullName: 'Wrong Name', usn }])

    await signInWithCode(page, REPORTER)
    await page.waitForURL('**/welcome')
    await page.getByRole('button', { name: 'Not you? Report it' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Report and sign out' }).click()
    await page.waitForURL('**/login')
    await expect(page.getByRole('heading', { name: "Thanks. You're signed out." })).toBeVisible()

    await signInWithCode(page, REPORTER)
    await expect(page.getByRole('heading', { name: 'Your request is still waiting' })).toBeVisible()
  })
})
