import { test, expect, type APIRequestContext } from '@playwright/test'
import { getDatabaseURL, getSchemaClient, replaceActivationToken } from '../helpers/db'
import {
  bootstrapAdmin,
  setupAdmin,
  loginViaUI,
  cleanupTestUsers,
  getUserIdByEmail,
  adminApproveUser,
  expectHome,
} from '../helpers/auth'

const TS = Date.now()
const YEAR = String(new Date().getFullYear()).slice(2)
const STUDENT = {
  email: `e2e-reader-${TS}@test.com`,
  password: 'E2EPass123',
  full_name: 'Priya Kumar',
  usn: `4MN${YEAR}CS${String(TS).slice(-3)}`,
}
// Older notices that push the feed past one page (20), so scrolling must load more.
const CIRCULARS = 22
const DAY = 24 * 60 * 60 * 1000

async function departmentId(code: string): Promise<string> {
  const client = await getSchemaClient()
  try {
    const result = await client.query('SELECT id FROM departments WHERE code = $1', [code])
    return result.rows[0].id
  } finally {
    await client.end()
  }
}

async function post(request: APIRequestContext, token: string, data: Record<string, unknown>) {
  const res = await request.post('/api/v1/announcements', {
    data,
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`Posting failed (${res.status()}): ${await res.text()}`)
}

test.describe('Reading notices', () => {
  let dbURL: string

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    const adminToken = await setupAdmin(request, dbURL, admin)

    // The student joins through the real access request, so they have a
    // Student identity in CS like any student would.
    const access = await request.post('/api/v1/auth/request-access', {
      data: { ...STUDENT, department_code: 'CS' },
    })
    expect(access.ok()).toBeTruthy()
    const userId = await getUserIdByEmail(dbURL, STUDENT.email)
    await adminApproveUser(request, adminToken, userId)
    const activation = await request.post('/api/v1/auth/activate', {
      data: { token: await replaceActivationToken(userId), password: STUDENT.password },
    })
    expect(activation.ok()).toBeTruthy()

    const cs = await departmentId('CS')
    const ec = await departmentId('EC')
    for (let i = 1; i <= CIRCULARS; i++) {
      await post(request, adminToken, { title: `Circular ${i}`, body: `Circular number ${i}.`, category: 'official' })
    }
    await post(request, adminToken, {
      title: 'EC seminar hall booking',
      body: 'Only for EC students.',
      category: 'department',
      audience: [{ department_id: ec, role: 'student' }],
    })
    await post(request, adminToken, {
      title: 'CS lab 3 closed for maintenance on Monday',
      body: 'Use lab 2 instead.',
      category: 'department',
      audience: [{ department_id: cs, role: 'student' }],
    })
    await post(request, adminToken, {
      title: 'Mid-semester examination timetable published',
      body: 'The timetable is on the notice board.',
      category: 'official',
    })
    await post(request, adminToken, {
      title: 'Campus drive: registrations close Friday',
      body: 'A company is visiting campus for final-year students.\n\nRegister on the portal and upload your resume.',
      category: 'placement',
      expires_at: new Date(Date.now() + 4 * DAY).toISOString(),
    })
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, [STUDENT.email])
  })

  test('a student reads notices on desktop', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 })
    await loginViaUI(page, STUDENT.email)
    await expectHome(page)

    // Home shows the newest notices meant for this student, and nothing else.
    const latest = page.getByRole('region', { name: 'Latest notices' })
    await expect(latest.getByRole('link', { name: /Campus drive/ })).toBeVisible()
    await expect(latest.getByRole('link', { name: /CS lab 3 closed/ })).toBeVisible()
    await expect(page.getByText('EC seminar hall booking')).toHaveCount(0)
    await expect(page.getByRole('navigation', { name: 'Main' }).getByText('Student', { exact: true })).toBeVisible()

    // The feed loads older notices as the reader scrolls.
    await page.getByRole('link', { name: 'All notices →' }).click()
    await page.waitForURL('**/notices')
    await expect(page.getByRole('heading', { level: 1, name: 'Notices' })).toBeVisible()
    await expect(page.getByRole('link', { name: /Circular 1\b/ })).toHaveCount(0)
    await page.getByText("You're all caught up.").or(page.getByRole('button', { name: /older notices/ })).scrollIntoViewIfNeeded()
    await expect(page.getByRole('link', { name: /Circular 1\b/ })).toBeVisible()
    await expect(page.getByText("You're all caught up.")).toBeVisible()

    // The category filter shows one category and says which is chosen.
    await page.getByRole('navigation', { name: 'Filter by category' }).getByRole('link', { name: 'Placement' }).click()
    await expect(page).toHaveURL(/category=placement/)
    await expect(page.getByRole('link', { name: 'Placement', exact: true })).toHaveAttribute('aria-current', 'true')
    await expect(page.getByRole('link', { name: /Campus drive/ })).toBeVisible()
    await expect(page.getByRole('link', { name: /Mid-semester/ })).toHaveCount(0)

    // The detail view shows the whole notice, who sent it to whom, and its expiry.
    await page.getByRole('link', { name: /Campus drive/ }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'Campus drive: registrations close Friday' })).toBeVisible()
    await expect(page.getByText('Register on the portal and upload your resume.')).toBeVisible()
    const about = page.getByRole('complementary')
    await expect(about.getByText('Whole college')).toBeVisible()
    await expect(about.getByText(/in 4 days/i)).toBeVisible()

    await page.getByRole('link', { name: 'All notices' }).click()
    await page.waitForURL('**/notices')
  })

  test('a student reads notices on a phone', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, STUDENT.email)
    await expectHome(page)

    // Phones get bottom navigation instead of the sidebar.
    const tabs = page.getByRole('navigation', { name: 'Main' })
    await expect(tabs).toHaveCount(1)
    await expect(tabs.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')

    await tabs.getByRole('link', { name: 'Notices' }).click()
    await page.waitForURL('**/notices')
    await expect(tabs.getByRole('link', { name: 'Notices' })).toHaveAttribute('aria-current', 'page')

    await page.getByRole('navigation', { name: 'Filter by category' }).getByRole('link', { name: 'Dept' }).click()
    await page.getByRole('link', { name: /CS lab 3 closed/ }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'CS lab 3 closed for maintenance on Monday' })).toBeVisible()
    await expect(page.getByRole('article').getByText('CS students')).toBeVisible()
    await expect(page.getByText('Use lab 2 instead.')).toBeVisible()
  })

  test('a notice that is not for the reader is not shown', async ({ page }) => {
    await loginViaUI(page, STUDENT.email)
    await page.goto('/notices/00000000-0000-0000-0000-000000000000')
    await expect(page.getByRole('heading', { name: "This notice isn't available" })).toBeVisible()
  })
})
