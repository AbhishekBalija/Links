import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// EC is used by no other spec, so these proposals reach only this spec's HOD.
async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

// A date a few days ahead, as the date input wants it.
function daysAhead(days: number) {
  const d = new Date(Date.now() + days * 24 * 3600 * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function propose(request: APIRequestContext, token: string, body: Record<string, unknown>) {
  const res = await request.post('/api/v1/events', { data: body, headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok()) throw new Error(`Proposing failed (${res.status()}): ${await res.text()}`)
  return (await res.json()).data.id as string
}

test.describe('Proposing events', () => {
  let faculty: { email: string; userId: string }
  let hod: { email: string; userId: string }

  test.beforeAll(async ({ request }) => {
    faculty = await seedMember(request, { role: 'faculty', fullName: 'Meera Faculty', department: 'EC' })
    hod = await seedMember(request, { role: 'hod', fullName: 'Asha HOD', department: 'EC' })
  })

  test('faculty propose an event, the HOD asks for changes, and they resubmit', async ({ page, request }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, faculty.email)
    await page.getByRole('link', { name: 'My posts' }).click()
    await page.getByRole('button', { name: 'New' }).click()
    await page.getByRole('link', { name: /An event/ }).click()
    await expect(page.getByRole('heading', { name: 'Propose an event' })).toBeVisible()

    // Submitting too early lists what's missing and keeps what was typed.
    await page.getByLabel('Where').fill('EC Seminar Hall')
    await page.getByRole('button', { name: 'Submit for approval' }).click()
    await expect(page.getByRole('alert')).toContainText('4 things to fix')
    await expect(page.getByText('Give the event a title.')).toBeVisible()
    await expect(page.getByLabel('Where')).toHaveValue('EC Seminar Hall')

    await page.getByRole('button', { name: 'Talk', exact: true }).click()
    await page.getByLabel('Title').fill('Guest talk on radio design')
    await page.getByLabel('Starts date').fill(daysAhead(5))
    await page.getByLabel('Starts time').fill('14:30')
    await page.getByLabel('Ends time').fill('16:00')
    await expect(page.getByText('1 h 30 min')).toBeVisible()
    // The usual case needs no choice: the proposer's department's students.
    await expect(page.getByRole('radio', { name: 'EC students' })).toBeChecked()
    await expect(page.getByText("Goes to the EC HOD, then the principal. It's published once both approve.")).toBeVisible()

    await page.getByRole('button', { name: 'Submit for approval' }).click()
    await page.waitForURL(/\/mine\/events\/[0-9a-f-]+$/)
    await expect(page.getByRole('heading', { level: 1, name: 'Guest talk on radio design' })).toBeVisible()
    await expect(page.getByText('With the EC HOD', { exact: true })).toBeVisible()
    const id = page.url().split('/').pop() ?? ''

    const hodToken = await loginViaAPI(request, hod.email)
    const asked = await request.patch(`/api/v1/events/${id}/hod-review`, {
      data: { decision: 'request_changes', note: 'Please start at 3 pm; labs run until 2:45.' },
      headers: { Authorization: `Bearer ${hodToken}` },
    })
    expect(asked.ok()).toBeTruthy()

    // My posts opens on what needs the author.
    await page.goto('/mine')
    await expect(page.getByRole('button', { name: /Needs you/, pressed: true })).toBeVisible()
    await page.getByRole('link', { name: /Guest talk on radio design/ }).click()
    await expect(page.getByText('Please start at 3 pm; labs run until 2:45.')).toBeVisible()
    await page.getByRole('link', { name: 'Edit and resubmit' }).click()

    await expect(page.getByRole('note')).toContainText('Please start at 3 pm')
    await page.getByLabel('Starts time').fill('15:00')
    await expect(page.getByText('Goes back to the EC HOD, then the principal.')).toBeVisible()
    await page.getByRole('button', { name: 'Resubmit' }).click()
    await page.waitForURL(/\/mine\/events\/[0-9a-f-]+$/)
    await expect(page.getByText('With the EC HOD', { exact: true })).toBeVisible()
  })

  test('a draft is saved on a phone, then deleted after asking', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, faculty.email)
    await page.goto('/mine/events/new')
    await page.getByRole('button', { name: 'Workshop', exact: true }).click()
    await page.getByLabel('Title').fill('Soldering workshop')
    await page.getByLabel('Starts date').fill(daysAhead(9))
    await page.getByLabel('Starts time').fill('10:00')
    await page.getByLabel('Ends time').fill('12:00')
    await page.getByLabel('Where').fill('EC Lab 2')
    await page.getByRole('button', { name: 'Save draft' }).click()

    await page.waitForURL('**/mine?status=draft')
    await page.getByRole('link', { name: /Soldering workshop/ }).click()
    await expect(page.getByText('Only you can see it.')).toBeVisible()
    await page.getByRole('button', { name: 'Delete draft' }).click()
    await expect(page.getByText('Delete this draft?')).toBeVisible()
    await page.getByRole('button', { name: 'Delete draft' }).click()
    await page.waitForURL('**/mine?status=draft')
    await expect(page.getByText('No drafts.')).toBeVisible()
  })

  test('cancelling a waiting proposal needs a reason', async ({ page, request }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    const token = await loginViaAPI(request, faculty.email)
    const ec = await departmentId('EC')
    const starts = new Date(Date.now() + 6 * 24 * 3600 * 1000)
    const id = await propose(request, token, {
      title: 'Circuits quiz',
      description: '',
      event_type: 'competition',
      department_id: ec,
      location: 'EC Block',
      starts_at: starts.toISOString(),
      ends_at: new Date(starts.getTime() + 3600 * 1000).toISOString(),
      audience: [{ department_id: ec, role: 'student' }],
    })

    await loginViaUI(page, faculty.email)
    await page.goto(`/mine/events/${id}`)
    await page.getByRole('button', { name: 'Cancel proposal' }).click()
    const dialog = page.getByRole('dialog', { name: 'Cancel this proposal?' })
    await dialog.getByRole('button', { name: 'Cancel proposal' }).click()
    await expect(dialog.getByText('Say why, in a sentence.')).toBeVisible()
    await dialog.getByLabel('Reason (the reviewers see this)').fill('The quiz master is away that week.')
    await dialog.getByRole('button', { name: 'Cancel proposal' }).click()

    await page.waitForURL('**/mine?status=ended')
    await expect(page.getByRole('link', { name: /Circuits quiz/ })).toContainText('Cancelled')
  })
})

test.describe('My posts', () => {
  test('Ended says so when rejected proposals fail to load, instead of leaving them out', async ({ page, request }) => {
    const faculty = await seedMember(request, { role: 'faculty', fullName: 'Ravi Faculty', department: 'EC' })
    const token = await loginViaAPI(request, faculty.email)
    const starts = new Date(Date.now() + 6 * 24 * 3600 * 1000)
    await propose(request, token, {
      title: 'A draft so My posts is not empty',
      event_type: 'other',
      department_id: await departmentId('EC'),
      location: 'EC Block',
      starts_at: starts.toISOString(),
      ends_at: new Date(starts.getTime() + 3600 * 1000).toISOString(),
      audience: [],
      draft: true,
    })
    await loginViaUI(page, faculty.email)
    await page.route('**/api/v1/events/mine?*status=attention*', (route) => route.fulfill({ status: 500, body: '{"error":{"code":"INTERNAL","message":"x"}}' }))
    await page.goto('/mine?status=ended')
    await expect(page.getByText('Your posts could not be loaded.')).toBeVisible()
  })
})
