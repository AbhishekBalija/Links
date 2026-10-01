import { test, expect } from '@playwright/test'
import { loginViaUI, seedMember } from '../helpers/auth'

function dateInDays(days: number) {
  const d = new Date(Date.now() + days * 24 * 3600 * 1000)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

test.describe('Placement', () => {
  test('the placement officer drafts, publishes and closes an opportunity that the right students see', async ({ page, browser, request }) => {
    // AD batch 2025 is used by no other spec, so only this student is eligible.
    const officer = await seedMember(request, { role: 'placement_officer', fullName: 'Nisha Officer' })
    const student = await seedMember(request, { role: 'student', fullName: 'Tara Student', department: 'AD', batch: 2025 })

    await page.setViewportSize({ width: 1440, height: 960 })
    await loginViaUI(page, officer.email)
    await page.getByRole('navigation', { name: 'Main' }).first().getByRole('link', { name: 'Placement' }).click()
    await page.waitForURL('**/placement')
    await page.getByRole('link', { name: 'New opportunity' }).first().click()

    await page.getByRole('button', { name: 'Save draft' }).click()
    await expect(page.getByRole('alert').first()).toContainText('things to fix')
    await expect(page.getByText('Write the role.')).toBeVisible()

    await page.getByRole('group', { name: 'Type' }).getByRole('button', { name: 'Job' }).click()
    await page.getByLabel('Role', { exact: true }).fill('Robotics trainee')
    await page.getByLabel('Company', { exact: true }).fill('Tessel Robotics')
    await page.getByLabel('Apply by date').fill(dateInDays(12))
    await page.getByRole('radio', { name: 'Choose departments and batches' }).check()
    await page.getByRole('group', { name: 'Departments' }).getByRole('button', { name: 'AD' }).click()
    await page.getByRole('group', { name: 'Batch' }).getByRole('button', { name: '2025' }).click()
    await expect(page.getByText('Shows in Jobs for AD students, batch 2025.')).toBeVisible()
    await page.getByRole('button', { name: 'Save draft' }).click()

    await page.waitForURL(/\/placement\/[0-9a-f-]+$/)
    await expect(page.getByRole('heading', { name: 'Robotics trainee' })).toBeVisible()
    await expect(page.getByText('Draft', { exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Publish', exact: true }).click()
    const publish = page.getByRole('dialog', { name: 'Publish to AD students, batch 2025?' })
    await publish.getByRole('button', { name: 'Publish', exact: true }).click()
    await expect(page.getByText('Open', { exact: true })).toBeVisible()

    const studentContext = await browser.newContext({ viewport: { width: 1440, height: 960 } })
    const studentPage = await studentContext.newPage()
    await loginViaUI(studentPage, student.email)
    await studentPage.goto('/jobs')
    await expect(studentPage.getByRole('link', { name: /Robotics trainee/ })).toContainText('Tessel Robotics')
    await studentContext.close()

    await page.getByRole('button', { name: 'Close early' }).click()
    const close = page.getByRole('dialog', { name: 'Close applications now?' })
    await close.getByRole('button', { name: 'Close applications' }).click()
    await expect(page.getByText('Closed early', { exact: false }).first()).toBeVisible()

    await page.getByRole('link', { name: 'Placement', exact: true }).first().click()
    await page.getByRole('link', { name: 'Closed', exact: true }).click()
    await expect(page.getByRole('link', { name: /Robotics trainee/ })).toBeVisible()
  })

  test('on a phone, the officer has a Placement tab and is asked before losing a half-written opportunity', async ({ page, request }) => {
    const officer = await seedMember(request, { role: 'placement_officer', fullName: 'Nisha Officer' })
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, officer.email)
    await page.getByRole('navigation', { name: 'Main' }).last().getByRole('link', { name: 'Placement' }).click()
    await page.getByRole('link', { name: 'New opportunity' }).first().click()
    await page.getByLabel('Role', { exact: true }).fill('Half-written role')
    await page.getByRole('link', { name: 'Placement' }).first().click()
    const leave = page.getByRole('dialog', { name: 'Keep this as a draft?' })
    await expect(leave).toBeVisible()
    await leave.getByRole('button', { name: 'Keep editing' }).click()
    await expect(page.getByLabel('Role', { exact: true })).toHaveValue('Half-written role')
  })
})
