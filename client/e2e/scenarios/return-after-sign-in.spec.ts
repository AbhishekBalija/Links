import { test, expect } from '@playwright/test'
import { getDatabaseURL } from '../helpers/db'
import { bootstrapAdmin, freeUSN, importStudents, lastCode, loginViaAPI, loginViaUI, seedMember, setupAdmin, typeCode } from '../helpers/auth'

// Getting back to where you were (#212): a link opened while signed out
// lands there after signing in, and a session that ends says so.

test.describe('Coming back after signing in', () => {
  test('a shared link opened while signed out lands there after signing in, even through the first sign-in', async ({ page, request }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    await setupAdmin(request, getDatabaseURL(), admin)
    const email = `e2e-shared-link-${Date.now()}@test.com`
    await importStudents(request, await loginViaAPI(request, admin.email), [{ email, fullName: 'Shared Link', usn: await freeUSN('CS', 2023) }])

    await page.goto('/events?when=past')
    await page.waitForURL('**/login')
    await page.getByLabel('Email').fill(email)
    await page.getByRole('button', { name: 'Email me a code' }).click()
    await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()
    await typeCode(page, await lastCode(request, email))

    // A reload doesn't lose the first sign-in step.
    await page.waitForURL('**/welcome')
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Welcome to Links, Shared' })).toBeVisible()
    await page.getByRole('button', { name: "Yes, that's me" }).click()
    await page.waitForURL('**/events?when=past')
  })

  test('a session that ends on its own says so, and signing in again goes back', async ({ page, request }) => {
    const member = await seedMember(request, { role: 'faculty', fullName: 'Session Ends', department: 'ME' })
    await loginViaUI(page, member.email)
    await page.goto('/events')
    await expect(page.getByRole('heading', { level: 1, name: 'Events' })).toBeVisible()

    // The server stops renewing the session (a role ended, an account was
    // paused); access tokens last seconds in this suite.
    await page.route('**/api/v1/auth/refresh', (route) => route.fulfill({ status: 401, contentType: 'application/json', body: '{"error":{"code":"UNAUTHENTICATED","message":"session ended"}}' }))
    await page.waitForTimeout(11_000)
    await page.getByRole('link', { name: 'Notices' }).first().click()

    await page.waitForURL('**/login')
    await expect(page.getByText('Your session ended, so you were signed out.')).toBeVisible()

    await page.unroute('**/api/v1/auth/refresh')
    await page.getByLabel('Email').fill(member.email)
    await page.getByRole('button', { name: 'Email me a code' }).click()
    await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()
    await typeCode(page, await lastCode(request, member.email))
    await page.waitForURL('**/notices')
  })
})
