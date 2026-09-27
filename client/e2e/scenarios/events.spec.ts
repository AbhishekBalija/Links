import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// ME is used by no other spec, so these events reach only this student.
async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

async function publish(request: APIRequestContext, token: string, body: Record<string, unknown>) {
  const res = await request.post('/api/v1/events', { data: body, headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok()) throw new Error(`Publishing failed (${res.status()}): ${await res.text()}`)
  return (await res.json()).data.id as string
}

test.describe('Events', () => {
  test('a student finds an event, says they are going, and sees it in their list', async ({ page, request }) => {
    const student = await seedMember(request, { role: 'student', fullName: 'Ravi Student', department: 'ME', batch: 2024 })
    const principal = await seedMember(request, { role: 'principal', fullName: 'Principal Rao' })
    const token = await loginViaAPI(request, principal.email, principal.password)
    const me = await departmentId('ME')
    const starts = new Date(Date.now() + 3 * 24 * 3600 * 1000)
    starts.setHours(14, 30, 0, 0)
    const ends = new Date(starts.getTime() + 90 * 60 * 1000)
    await publish(request, token, {
      title: 'Guest talk on engines',
      description: 'How engines are tested.',
      event_type: 'talk',
      location: 'ME Seminar Hall',
      department_id: me,
      starts_at: starts.toISOString(),
      ends_at: ends.toISOString(),
      capacity: 50,
      audience: [{ department_id: me }],
    })

    await loginViaUI(page, student.email, student.password)
    await page.getByRole('navigation', { name: 'Main' }).first().getByRole('link', { name: 'Events' }).click()
    await page.waitForURL('**/events')
    const row = page.getByRole('link', { name: /Guest talk on engines/ })
    await expect(row).toContainText('50 of 50 seats left')

    await row.click()
    await expect(page.getByRole('heading', { name: 'Guest talk on engines' })).toBeVisible()
    await expect(page.getByText('Are you going?')).toBeVisible()
    await page.getByRole('group', { name: 'Your answer' }).getByRole('button', { name: 'Going' }).click()
    await expect(page.getByText("You're going. Choose another answer to change it.")).toBeVisible()

    await page.getByRole('link', { name: 'Events', exact: true }).first().click()
    await page.waitForURL('**/events')
    await expect(page.getByRole('link', { name: /Guest talk on engines/ })).toContainText("You're going")

    await page.getByRole('link', { name: 'Going', exact: true }).click()
    await expect(page.getByRole('link', { name: /Guest talk on engines/ })).toBeVisible()
  })
})
