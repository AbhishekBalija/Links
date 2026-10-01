import { test, expect } from '@playwright/test'
import { loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// A student finds a staff member by searching People, opens their profile,
// and moves on to that Department's page.
test.describe('People', () => {
  test('search, open a profile and its Department', async ({ page, request }) => {
    const student = await seedMember(request, { role: 'student', fullName: 'Ravi Student', department: 'ME', batch: 2024 })
    const hod = await seedMember(request, { role: 'hod', fullName: 'Nandini Rao', department: 'ME' })
    const client = await getSchemaClient()
    try {
      // Listed members are verified, as an approved or imported account is.
      await client.query(`UPDATE users SET is_verified = true WHERE id = ANY($1)`, [[student.userId, hod.userId]])
      await client.query(`UPDATE profiles SET headline = 'Head of ME' WHERE user_id = $1`, [hod.userId])
    } finally {
      await client.end()
    }

    await loginViaUI(page, student.email)
    await page.getByRole('navigation', { name: 'Main' }).first().getByRole('link', { name: 'People' }).click()
    await page.waitForURL('**/people')

    await page.getByRole('searchbox', { name: 'Search people' }).fill('nandni')
    const match = page.getByRole('link', { name: /Nandini Rao/ })
    await expect(match).toBeVisible()
    await expect(page.getByText(/match(es)? for “nandni”/)).toBeVisible()

    await match.click()
    await expect(page.getByRole('heading', { name: 'Nandini Rao' })).toBeVisible()
    await expect(page.getByText('HOD ·')).toBeVisible()

    // ME is used by no other spec, so Nandini is its only HOD.
    await page.locator('main a[href="/departments/ME"]').first().click()
    await page.waitForURL('**/departments/ME')
    await expect(page.getByText('Head of department')).toBeVisible()
    await expect(page.getByRole('link', { name: /Nandini Rao/ })).toBeVisible()

    // Back returns to the search, filters and all.
    await page.getByRole('link', { name: 'People', exact: true }).last().click()
    await page.waitForURL('**/people**')
  })
})

test.describe('People after editing a profile', () => {
  test('a new headline shows in People straight away, without reloading', async ({ page, request }) => {
    // CV batch 2021 is used by no other People spec.
    const student = await seedMember(request, { role: 'student', fullName: 'Kavya Headline', department: 'CV', batch: 2021 })
    const client = await getSchemaClient()
    try {
      await client.query(`UPDATE users SET is_verified = true WHERE id = $1`, [student.userId])
    } finally {
      await client.end()
    }

    await page.setViewportSize({ width: 1440, height: 960 })
    await loginViaUI(page, student.email)
    const nav = page.getByRole('navigation', { name: 'Main' }).first()
    await nav.getByRole('link', { name: 'People' }).click()
    await expect(page.getByRole('link', { name: /Kavya Headline/ })).toBeVisible()

    // Only in-app links from here: a reload would hide a stale cache.
    await nav.getByRole('link', { name: 'Profile' }).click()
    await page.getByRole('link', { name: 'Edit profile' }).click()
    await page.locator('#headline').fill('Bridges and concrete')
    await page.getByRole('button', { name: 'Save' }).click()
    await page.waitForURL('**/profile')
    await nav.getByRole('link', { name: 'People' }).click()
    await expect(page.getByRole('link', { name: /Kavya Headline/ })).toContainText('Bridges and concrete')
  })
})
