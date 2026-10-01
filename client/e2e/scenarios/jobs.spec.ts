import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// AD is used by no other spec, so these Opportunities reach only these students.
async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

async function publishOpportunity(request: APIRequestContext, token: string, body: Record<string, unknown>) {
  const headers = { Authorization: `Bearer ${token}` }
  const created = await request.post('/api/v1/opportunities', { data: body, headers })
  if (!created.ok()) throw new Error(`Saving the draft failed (${created.status()}): ${await created.text()}`)
  const id = (await created.json()).data.id as string
  const published = await request.post(`/api/v1/opportunities/${id}/publish`, { headers })
  if (!published.ok()) throw new Error(`Publishing failed (${published.status()}): ${await published.text()}`)
  return id
}

function inDays(days: number) {
  return new Date(Date.now() + days * 24 * 3600 * 1000).toISOString()
}

test.describe('Jobs', () => {
  test('a student applies to a job in LINKS, sees it under Applied, and withdraws', async ({ page, request }) => {
    const student = await seedMember(request, { role: 'student', fullName: 'Meera Student', department: 'AD', batch: 2023 })
    const officer = await seedMember(request, { role: 'placement_officer', fullName: 'Nisha Officer' })
    const token = await loginViaAPI(request, officer.email, officer.password)
    const ad = await departmentId('AD')
    await publishOpportunity(request, token, {
      opportunity_type: 'job',
      title: 'Graduate Engineer Trainee',
      company: 'Acme Systems',
      description: 'Six months of training, then a product team.',
      location: 'Mysuru',
      compensation: '4.5 LPA',
      apply_by: inDays(10),
      application_mode: 'internal',
      eligibility: [{ department_id: ad, batch_year: 2023, role: 'student' }],
    })

    await page.setViewportSize({ width: 1440, height: 960 })
    await loginViaUI(page, student.email, student.password)
    await page.getByRole('navigation', { name: 'Main' }).first().getByRole('link', { name: 'Jobs' }).click()
    await page.waitForURL('**/jobs')
    const row = page.getByRole('link', { name: /Graduate Engineer Trainee/ })
    await expect(row).toContainText('Acme Systems · Mysuru · 4.5 LPA')
    await expect(row).toContainText('10 days left')

    await row.click()
    await expect(page.getByRole('heading', { name: 'Graduate Engineer Trainee' })).toBeVisible()
    await page.getByRole('button', { name: 'Apply', exact: true }).click()
    const confirm = page.getByRole('dialog', { name: 'Apply to Acme Systems?' })
    await expect(confirm).toContainText("you can't apply to this one again")
    await confirm.getByRole('button', { name: 'Apply', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'You applied' })).toBeVisible()

    await page.getByRole('link', { name: 'Jobs', exact: true }).first().click()
    await page.getByRole('link', { name: 'Applied', exact: true }).click()
    await expect(page.getByRole('link', { name: /Graduate Engineer Trainee/ })).toContainText('Applied')

    await page.getByRole('link', { name: /Graduate Engineer Trainee/ }).click()
    await page.getByRole('button', { name: 'Withdraw application' }).click()
    const withdraw = page.getByRole('dialog', { name: 'Withdraw your application?' })
    await withdraw.getByRole('button', { name: 'Withdraw', exact: true }).click()
    await expect(page.getByText("You withdrew", { exact: false })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Apply', exact: true })).toHaveCount(0)
  })

  test('on a phone, a student records that they applied on the company site', async ({ page, request }) => {
    const student = await seedMember(request, { role: 'student', fullName: 'Kiran Student', department: 'AD', batch: 2024 })
    const officer = await seedMember(request, { role: 'placement_officer', fullName: 'Nisha Officer' })
    const token = await loginViaAPI(request, officer.email, officer.password)
    const ad = await departmentId('AD')
    await publishOpportunity(request, token, {
      opportunity_type: 'internship',
      title: 'Data analyst intern',
      company: 'Kaveri Analytics',
      description: 'Ten weeks with the analytics team.',
      apply_by: inDays(2),
      application_mode: 'external',
      external_url: 'https://example.com/apply',
      eligibility: [{ department_id: ad, batch_year: 2024, role: 'student' }],
    })

    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, student.email, student.password)
    const tabs = page.getByRole('navigation', { name: 'Main' }).last()
    await expect(tabs.getByRole('link', { name: 'People' })).toHaveCount(0)
    await tabs.getByRole('link', { name: 'Jobs' }).click()
    await page.getByRole('link', { name: /Data analyst intern/ }).click()

    await expect(page.getByRole('link', { name: /Open company site/ })).toHaveAttribute('href', 'https://example.com/apply')
    await page.getByRole('button', { name: 'I applied there' }).click()
    const ask = page.getByRole('dialog', { name: 'Did you apply on their site?' })
    await ask.getByRole('button', { name: 'Yes, I applied' }).click()
    await expect(page.getByText('Applied on their site')).toBeVisible()
  })
})
