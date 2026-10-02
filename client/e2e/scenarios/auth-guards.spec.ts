import { test, expect } from '@playwright/test'
import { getDatabaseURL, getSchemaClient } from '../helpers/db'
import { bootstrapAdmin, setupAdmin, loginViaUI, cleanupTestUsers, seedMember, expectHome } from '../helpers/auth'

const emails: string[] = []

test.describe('Auth Guards', () => {
  let dbURL: string

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    await setupAdmin(request, dbURL, admin)
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, emails)
  })

  test('Protected route redirects to the sign-in screen when signed out', async ({ page }) => {
    await page.goto('/')
    await page.waitForURL('**/login')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('Sign in')
  })

  test('Old password links lead to the sign-in screen', async ({ page }) => {
    for (const path of ['/access-request', '/activate?token=x']) {
      await page.goto(path)
      await page.waitForURL('**/login')
      await expect(page.getByRole('heading', { level: 1 })).toHaveText('Sign in')
    }
  })

  test('Logout clears session and redirects to sign-in', async ({ page, request }) => {
    const student = await seedMember(request, { role: 'student', fullName: 'Guard Test', department: 'CS', batch: 2024 })
    emails.push(student.email)

    await loginViaUI(page, student.email)
    await expectHome(page)

    // A member with a role can't stay on the no-role screen.
    await page.goto('/account-pending')
    await page.waitForURL((url) => url.pathname === '/')
    await expectHome(page)

    await page.getByRole('button', { name: 'Log out' }).first().click()
    await page.waitForURL('**/login')
    await expect(page.getByText("You're signed out on this device.")).toBeVisible()

    await page.goto('/')
    await page.waitForURL('**/login')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('Sign in')
  })

  test('A member with no role lands on the no-role screen and can sign out', async ({ page, request }) => {
    const member = await seedMember(request, { role: 'student', fullName: 'Zero Role User', department: 'CS', batch: 2024 })
    emails.push(member.email)
    // Their only role ends, as when an admin removes it.
    const client = await getSchemaClient()
    try {
      await client.query(`DELETE FROM role_assignments WHERE user_id = $1`, [member.userId])
    } finally {
      await client.end()
    }

    await loginViaUI(page, member.email)
    await page.waitForURL('**/account-pending')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('Your account has no role yet')

    await page.getByRole('button', { name: 'Sign out' }).click()
    await page.waitForURL('**/login')
  })
})
