import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// The Home of people who post and propose (#142): faculty get one built
// around their own posts; student coordinators keep the student Home with
// New added.

async function departmentId(code: string) {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT id FROM departments WHERE code = $1', [code])).rows[0].id as string
  } finally {
    await client.end()
  }
}

async function call(request: APIRequestContext, method: 'post' | 'patch', url: string, token: string, data: Record<string, unknown>) {
  const res = await request[method](url, { data, headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok()) throw new Error(`${method} ${url} failed (${res.status()}): ${await res.text()}`)
  return (await res.json()).data.id as string
}

test.describe('Home for authors', () => {
  test('faculty start with what they can do, then see what came back and what waits', async ({ page, request }) => {
    const faculty = await seedMember(request, { role: 'faculty', fullName: 'Kiran Hegde', department: 'EC' })
    const hod = await seedMember(request, { role: 'hod', fullName: 'Lakshmi Rao', department: 'EC' })
    const ec = await departmentId('EC')
    await page.setViewportSize({ width: 1280, height: 900 })

    // First time: what they can do, with a way to start each.
    await loginViaUI(page, faculty.email)
    await expect(page.getByRole('heading', { name: 'Post to your classes' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Write an announcement' })).toHaveAttribute('href', '/mine/new')
    await expect(page.getByRole('link', { name: 'Propose an event' })).toHaveAttribute('href', '/mine/events/new')

    // One announcement sent back with a note, one event waiting on the HOD.
    const token = await loginViaAPI(request, faculty.email)
    const notice = await call(request, 'post', '/api/v1/announcements', token, {
      title: 'Lab 2 timings change',
      body: 'Details inside.',
      category: 'department',
      audience: [{ department_id: ec, role: 'student' }],
    })
    const starts = new Date(Date.now() + 6 * 24 * 3600 * 1000)
    await call(request, 'post', '/api/v1/events', token, {
      title: 'Antenna design workshop',
      description: '',
      event_type: 'workshop',
      department_id: ec,
      location: 'EC Block',
      starts_at: starts.toISOString(),
      ends_at: new Date(starts.getTime() + 3600 * 1000).toISOString(),
      audience: [{ department_id: ec, role: 'student' }],
    })
    const hodToken = await loginViaAPI(request, hod.email)
    await call(request, 'patch', `/api/v1/announcements/${notice}/approval`, hodToken, { decision: 'reject', note: 'Add the Lab 4 room number.' })

    await page.reload()
    await expect(page.getByText('One post was sent back with a note. One is waiting on others.')).toBeVisible()
    const needs = page.getByRole('region', { name: 'Needs you' })
    await expect(needs.getByText('Add the Lab 4 room number.')).toBeVisible()
    const waiting = page.getByRole('region', { name: 'Waiting on others' })
    await expect(waiting.getByText('Event · With the EC HOD')).toBeVisible()

    // New offers both kinds of post.
    await page.getByRole('button', { name: 'New' }).click()
    await expect(page.getByRole('link', { name: /An event/ })).toBeVisible()
    await page.keyboard.press('Escape')

    await needs.getByRole('link', { name: /Lab 2 timings change/ }).click()
    await expect(page).toHaveURL(new RegExp(`/mine/${notice}`))
  })

  test('on a phone, New sits in the top bar and opens a sheet', async ({ page, request }) => {
    const faculty = await seedMember(request, { role: 'faculty', fullName: 'Ravi Kumar', department: 'EC' })
    const token = await loginViaAPI(request, faculty.email)
    await call(request, 'post', '/api/v1/announcements', token, {
      title: 'Draft for later',
      body: 'Details inside.',
      category: 'department',
      audience: [{ department_id: await departmentId('EC'), role: 'student' }],
      draft: true,
    })
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, faculty.email)

    await expect(page.getByText('Nothing needs you right now.')).toBeVisible()
    await expect(page.getByRole('link', { name: /1 draft you haven't sent yet/ })).toBeVisible()
    await page.getByRole('button', { name: 'New' }).click()
    const sheet = page.getByRole('dialog', { name: 'What are you posting?' })
    await expect(sheet.getByRole('link', { name: /An announcement/ })).toBeVisible()
    await sheet.getByRole('button', { name: 'Cancel' }).click()
    await expect(sheet).toBeHidden()
  })

  test('a new student coordinator is welcomed once, then keeps the student Home with New', async ({ page, request }) => {
    const student = await seedMember(request, { role: 'student', fullName: 'Rohan Shetty', department: 'EC', batch: 2024 })
    const hod = await seedMember(request, { role: 'hod', fullName: 'Lakshmi Iyer', department: 'EC' })
    const client = await getSchemaClient()
    try {
      await client.query(
        `INSERT INTO role_assignments (user_id, role, scope_type, scope_id, assigned_by, starts_at) VALUES ($1, 'student_coordinator', 'department', $2, $3, NOW() - interval '1 minute')`,
        [student.userId, await departmentId('EC'), hod.userId],
      )
    } finally {
      await client.end()
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, student.email)

    const welcome = page.getByRole('dialog', { name: "You're a student coordinator now" })
    await expect(welcome).toContainText('Lakshmi Iyer made you a student coordinator for Electronics and Communication Engineering today.')
    await welcome.getByRole('button', { name: 'Got it' }).click()
    await expect(welcome).toBeHidden()

    await expect(page.getByRole('button', { name: 'New' })).toBeVisible()
    // Nothing of theirs is out yet, so no post sections take up the page.
    await expect(page.getByRole('region', { name: 'Needs you' })).toHaveCount(0)
    await expect(page.getByRole('heading', { name: 'Your announcements' })).toHaveCount(0)
    await expect(page.getByRole('heading', { name: 'Latest notices' })).toBeVisible()

    // Remembered on the account: not shown again, here or on a phone.
    await page.setViewportSize({ width: 390, height: 844 })
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Latest notices' })).toBeVisible()
    await expect(welcome).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'New' })).toBeVisible()

    // Their composer offers only what they may post: department notices to
    // their own department's students (#209).
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.goto('/mine/new')
    await expect(page.getByRole('radio', { name: /EC students/ })).toBeChecked()
    await expect(page.getByRole('radio', { name: /EC batch \d{4}/ })).toHaveCount(4)
    await expect(page.getByRole('radio', { name: /Whole college/ })).toHaveCount(0)
    await expect(page.getByText('Choose groups…')).toHaveCount(0)
    await expect(page.getByRole('button', { name: /Official/ })).toHaveCount(0)
  })
})
