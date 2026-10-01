import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// AI is used by no other spec, so these events reach only this spec's people.
async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

const auth = (token: string) => ({ Authorization: `Bearer ${token}` })

async function call(request: APIRequestContext, method: 'post' | 'patch', path: string, token: string, data: unknown) {
  const res = await request[method](path, { data, headers: auth(token) })
  if (!res.ok()) throw new Error(`${method} ${path} failed (${res.status()}): ${await res.text()}`)
  return (await res.json()).data
}

test.describe('Organising an event', () => {
  let faculty: { email: string; password: string }
  let eventId = ''

  test.beforeAll(async ({ request }) => {
    faculty = await seedMember(request, { role: 'faculty', fullName: 'Nisha Faculty', department: 'AI' })
    const hod = await seedMember(request, { role: 'hod', fullName: 'Vikram HOD', department: 'AI' })
    const principal = await seedMember(request, { role: 'principal', fullName: 'Principal Iyer' })
    const students = [
      await seedMember(request, { role: 'student', fullName: 'Tara Student', department: 'AI', batch: 2024 }),
      await seedMember(request, { role: 'student', fullName: 'Kabir Student', department: 'AI', batch: 2024 }),
    ]
    const ai = await departmentId('AI')
    const starts = new Date(Date.now() + 4 * 24 * 3600 * 1000)
    starts.setHours(11, 0, 0, 0)
    const created = await call(request, 'post', '/api/v1/events', await loginViaAPI(request, faculty.email), {
      title: 'Intro to neural networks',
      description: 'A hands-on session.',
      event_type: 'workshop',
      department_id: ai,
      location: 'AI Lab 1',
      starts_at: starts.toISOString(),
      ends_at: new Date(starts.getTime() + 2 * 3600 * 1000).toISOString(),
      capacity: 30,
      audience: [{ department_id: ai }],
    })
    eventId = created.id
    await call(request, 'patch', `/api/v1/events/${eventId}/hod-review`, await loginViaAPI(request, hod.email), { decision: 'approve', note: '' })
    await call(request, 'patch', `/api/v1/events/${eventId}/final-approval`, await loginViaAPI(request, principal.email), { decision: 'approve', note: '' })
    for (const student of students) {
      await call(request, 'post', `/api/v1/events/${eventId}/rsvp`, await loginViaAPI(request, student.email), { status: 'going' })
    }
  })

  test('the proposer sees who is coming, exports them and edits the details', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, faculty.email)
    await page.goto(`/events/${eventId}`)

    const coming = page.getByRole('complementary', { name: "Who's coming" })
    await expect(coming.getByText('Tara Student')).toBeVisible()
    await expect(coming.getByText('28 of 30 seats left')).toBeVisible()
    await expect(page.getByText('Published', { exact: false }).first()).toContainText('after the AI HOD and the principal approved it')

    const download = page.waitForEvent('download')
    await coming.getByRole('button', { name: 'Export CSV' }).click()
    expect((await download).suggestedFilename()).toBe('intro-to-neural-networks-answers.csv')

    await coming.getByRole('link', { name: /All 2 answers/ }).click()
    await expect(page.getByRole('heading', { name: "Who's coming" })).toBeVisible()
    await expect(page.getByText('Kabir Student')).toBeVisible()
    await page.getByRole('link', { name: 'Event', exact: true }).click()

    await page.getByRole('link', { name: 'Edit details' }).click()
    await expect(page.getByRole('heading', { name: 'Edit details' })).toBeVisible()
    await page.getByLabel('Seat limit').fill('1')
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(page.getByText("2 are going. A limit can't be lower than that.").last()).toBeVisible()
    await page.getByLabel('Seat limit').fill('40')
    await page.getByLabel('Where').fill('Main auditorium')
    await page.getByRole('button', { name: 'Save changes' }).click()
    await page.waitForURL(`**/events/${eventId}`)
    await expect(page.getByText('Main auditorium')).toBeVisible()
    await expect(page.getByText('38 of 40 seats left')).toBeVisible()
  })

  test('cancelling needs a reason, and the page then says it is cancelled', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, faculty.email)
    await page.goto(`/events/${eventId}`)
    await page.getByRole('button', { name: 'Cancel event' }).click()
    const dialog = page.getByRole('dialog', { name: 'Cancel this event?' })
    await expect(dialog).toContainText('The 2 people going see')
    await dialog.getByLabel('Reason (everyone invited sees this)').fill('The lab is closed for repairs.')
    await dialog.getByRole('button', { name: 'Cancel event' }).click()
    await expect(page.getByRole('status')).toContainText('The lab is closed for repairs.')
    await expect(page.getByRole('button', { name: 'Cancel event' })).toHaveCount(0)
  })
})
