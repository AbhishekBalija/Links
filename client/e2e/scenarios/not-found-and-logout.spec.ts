import { test, expect } from '@playwright/test'
import { loginViaUI, seedMember } from '../helpers/auth'

// #212: an unknown address says so instead of quietly landing on Home, and
// log out on phones is named, on your Profile, and asks first.

test.describe('Not found and phone log out', () => {
  test('an address LINKS does not have says so, with one way back', async ({ page, request }) => {
    const member = await seedMember(request, { role: 'faculty', fullName: 'Lost Link', department: 'ME' })
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, member.email)
    await page.goto('/jobz/old-link')
    await expect(page.getByRole('heading', { level: 1, name: "This page isn't here" })).toBeVisible()
    await page.getByRole('link', { name: 'Go to Home' }).click()
    await expect(page).toHaveURL(/\/$/)
  })

  test('on a phone, log out is on your Profile and asks first', async ({ page, request }) => {
    const member = await seedMember(request, { role: 'faculty', fullName: 'Phone Person', department: 'ME' })
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, member.email)
    // Home's top bar no longer signs you out with one tap.
    await expect(page.getByRole('button', { name: 'Log out' })).toHaveCount(0)

    await page.goto('/profile')
    await expect(page.getByText(member.email)).toBeVisible()
    await page.getByRole('button', { name: 'Log out' }).click()
    const sheet = page.getByRole('dialog', { name: 'Log out on this phone?' })
    await sheet.getByRole('button', { name: 'Cancel' }).click()
    await expect(sheet).toBeHidden()
    await expect(page).toHaveURL(/\/profile$/)

    await page.getByRole('button', { name: 'Log out' }).click()
    await sheet.getByRole('button', { name: 'Log out' }).click()
    await page.waitForURL('**/login')
    await expect(page.getByText("You're signed out on this device.")).toBeVisible()
  })
})
