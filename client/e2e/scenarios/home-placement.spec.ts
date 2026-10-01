import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

async function call(request: APIRequestContext, token: string, method: 'post' | 'patch', path: string, data?: unknown) {
  const res = await request[method](path, { data, headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok()) throw new Error(`${method} ${path} failed (${res.status()}): ${await res.text()}`)
  return (await res.json()).data
}

test.describe('Home and placement', () => {
  test('a student sees the jobs open to them on Home, and the placement officer sees the drives and what waits for review', async ({ page, browser, request }) => {
    // AD batch 2022 is used by no other spec.
    const officer = await seedMember(request, { role: 'placement_officer', fullName: 'Nisha Officer' })
    const student = await seedMember(request, { role: 'student', fullName: 'Ravi Student', department: 'AD', batch: 2022 })
    const token = await loginViaAPI(request, officer.email)
    const ad = await departmentId('AD')
    const opportunity = await call(request, token, 'post', '/api/v1/opportunities', {
      opportunity_type: 'internship',
      title: 'Robotics intern',
      company: 'Tessel Robotics',
      description: 'Build robot arms.',
      apply_by: new Date(Date.now() + 4 * 86400000).toISOString(),
      application_mode: 'internal',
      eligibility: [{ department_id: ad, batch_year: 2022, role: 'student' }],
    })
    await call(request, token, 'post', `/api/v1/opportunities/${opportunity.id}/publish`)

    const studentContext = await browser.newContext({ viewport: { width: 390, height: 844 } })
    const studentPage = await studentContext.newPage()
    await loginViaUI(studentPage, student.email)
    const jobs = studentPage.getByRole('region', { name: 'Open jobs for you' })
    await expect(jobs.getByRole('link', { name: /Robotics intern/ })).toBeVisible()
    await jobs.getByRole('link', { name: /Robotics intern/ }).click()
    await studentPage.getByRole('button', { name: 'Apply', exact: true }).click()
    await studentPage.getByRole('dialog').getByRole('button', { name: 'Apply', exact: true }).click()
    await expect(studentPage.getByText('Applied', { exact: true }).first()).toBeVisible()
    await studentContext.close()

    await page.setViewportSize({ width: 1440, height: 960 })
    await loginViaUI(page, officer.email)
    const placement = page.getByRole('region', { name: 'Open drives' })
    await expect(placement.getByRole('link', { name: /Robotics intern/ })).toContainText('1 to review')
    await expect(page.getByText(/waiting for review/).first()).toBeVisible()
    await page.getByRole('link', { name: /Review Tessel Robotics first/ }).click()
    await page.waitForURL(`**/placement/${opportunity.id}/applicants*`)
    await expect(page.getByRole('table')).toContainText('Ravi Student')
  })
})
