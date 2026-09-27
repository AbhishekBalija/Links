import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// CV is used by no other spec, so these proposals reach only this spec's HOD.
async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

async function propose(request: APIRequestContext, token: string, title: string, department: string) {
  const starts = new Date(Date.now() + 7 * 24 * 3600 * 1000)
  const res = await request.post('/api/v1/events', {
    data: {
      title,
      description: 'Bring your own laptop.',
      event_type: 'workshop',
      department_id: department,
      location: 'CV Drawing Hall',
      starts_at: starts.toISOString(),
      ends_at: new Date(starts.getTime() + 2 * 3600 * 1000).toISOString(),
      audience: [{ department_id: department, role: 'student' }],
    },
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`Proposing failed (${res.status()}): ${await res.text()}`)
}

test.describe('Reviewing events', () => {
  test('the HOD approves, then the principal asks for changes and publishes another', async ({ page, browser, request }) => {
    const faculty = await seedMember(request, { role: 'faculty', fullName: 'Leela Faculty', department: 'CV' })
    const hod = await seedMember(request, { role: 'hod', fullName: 'Mohan HOD', department: 'CV' })
    const principal = await seedMember(request, { role: 'principal', fullName: 'Principal Nair' })
    const cv = await departmentId('CV')
    const token = await loginViaAPI(request, faculty.email, faculty.password)
    await propose(request, token, 'AutoCAD basics', cv)
    await propose(request, token, 'Bridge design contest', cv)

    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, hod.email, hod.password)
    await page.goto('/approvals')
    const list = page.getByRole('navigation', { name: 'Waiting for you' })
    await list.getByRole('link', { name: /AutoCAD basics/ }).click()
    await expect(page.getByText('Approving sends it to the principal for final approval.')).toBeVisible()
    await page.getByRole('button', { name: 'Approve', exact: true }).click()
    await expect(page.getByRole('status')).toContainText('goes on to final approval')
    await list.getByRole('link', { name: /Bridge design contest/ }).click()
    await page.getByRole('button', { name: 'Approve', exact: true }).click()
    await expect(page.getByText('Nothing waiting for you')).toBeVisible()

    const context = await browser.newContext({ viewport: { width: 1280, height: 900 } })
    const principalPage = await context.newPage()
    await loginViaUI(principalPage, principal.email, principal.password)
    await principalPage.goto('/approvals')
    const queue = principalPage.getByRole('navigation', { name: 'Waiting for you' })
    await queue.getByRole('link', { name: /AutoCAD basics/ }).click()
    await expect(principalPage.getByText(/Mohan HOD, CV HOD approved it on .*\. Yours is the final approval\./)).toBeVisible()

    // Sending back always needs a note.
    await principalPage.getByRole('button', { name: 'Send back…' }).click()
    await principalPage.getByRole('radio', { name: /Ask for changes/ }).check()
    await principalPage.getByRole('button', { name: 'Send back', exact: true }).click()
    await expect(principalPage.getByText('Add a note so they know what to fix.')).toBeVisible()
    await principalPage.getByLabel(/What should they change/).fill('Add the software version you will use.')
    await principalPage.getByRole('button', { name: 'Send back', exact: true }).click()
    await expect(principalPage.getByRole('status')).toContainText('Sent back to Leela Faculty')

    // Final approval publishes, so it asks first.
    await queue.getByRole('link', { name: /Bridge design contest/ }).click()
    await principalPage.getByRole('button', { name: 'Approve and publish' }).click()
    await expect(principalPage.getByText('Publish it now?')).toBeVisible()
    await principalPage.getByRole('button', { name: 'Approve and publish' }).click()
    await expect(principalPage.getByRole('status')).toContainText('Published')
    await context.close()
  })
})
