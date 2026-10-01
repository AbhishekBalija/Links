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

test.describe('Applicants', () => {
  test('the placement officer searches applicants, shortlists one, is told when someone else changed a row first, and exports', async ({ page, request }) => {
    // AD batch 2026 is used by no other spec.
    const officer = await seedMember(request, { role: 'placement_officer', fullName: 'Nisha Officer' })
    const token = await loginViaAPI(request, officer.email, officer.password)
    const ad = await departmentId('AD')
    const opportunity = await call(request, token, 'post', '/api/v1/opportunities', {
      opportunity_type: 'job',
      title: 'Cloud trainee',
      company: 'Nimbus Cloud',
      description: 'Learn to run cloud systems.',
      apply_by: new Date(Date.now() + 10 * 86400000).toISOString(),
      application_mode: 'internal',
      eligibility: [{ department_id: ad, batch_year: 2026, role: 'student' }],
    })
    await call(request, token, 'post', `/api/v1/opportunities/${opportunity.id}/publish`)
    const applications: Record<string, string> = {}
    for (const name of ['Asha Applicant', 'Bala Applicant', 'Chitra Applicant']) {
      const student = await seedMember(request, { role: 'student', fullName: name, department: 'AD', batch: 2026 })
      const studentToken = await loginViaAPI(request, student.email, student.password)
      applications[name] = (await call(request, studentToken, 'post', `/api/v1/opportunities/${opportunity.id}/apply`)).id
    }

    await page.setViewportSize({ width: 1440, height: 960 })
    await loginViaUI(page, officer.email, officer.password)
    await page.goto(`/placement/${opportunity.id}`)
    await page.getByRole('link', { name: 'Open applicant list' }).click()
    await page.waitForURL('**/applicants')
    const table = page.getByRole('table')
    await expect(table.getByRole('row')).toHaveCount(4)

    await page.getByRole('searchbox', { name: 'Search applicants' }).fill('chitra')
    await expect(table.getByRole('row')).toHaveCount(2)
    await expect(table).toContainText('Chitra Applicant')
    await page.getByRole('searchbox', { name: 'Search applicants' }).fill('')
    await expect(table.getByRole('row')).toHaveCount(4)

    await page.getByRole('combobox', { name: 'Status for Asha Applicant' }).selectOption('shortlisted')
    await expect(page.getByRole('status').filter({ hasText: 'Asha Applicant is now Shortlisted' })).toBeVisible()
    await page.getByRole('link', { name: /^Shortlisted/ }).click()
    await expect(table.getByRole('row')).toHaveCount(2)
    await expect(table).toContainText('Asha Applicant')
    await page.getByRole('link', { name: 'All', exact: true }).click()
    await expect(table.getByRole('row')).toHaveCount(4)

    // Someone else in the office rejects Bala while this page still shows "To review".
    await call(request, token, 'patch', `/api/v1/opportunity-applications/${applications['Bala Applicant']}/status`, { from: 'applied', status: 'rejected' })
    await page.getByRole('combobox', { name: 'Status for Bala Applicant' }).selectOption('selected')
    await expect(page.getByRole('alert').filter({ hasText: 'Not changed' })).toContainText('Bala Applicant')
    await expect(page.getByRole('combobox', { name: 'Status for Bala Applicant' })).toHaveValue('rejected')

    await page.getByRole('button', { name: 'Export CSV' }).click()
    const download = page.waitForEvent('download')
    await page.getByRole('menuitem', { name: /Everyone/ }).click()
    expect((await download).suggestedFilename()).toBe('nimbus-cloud-cloud-trainee-applicants.csv')
  })
})
