import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember, expectHome } from '../helpers/auth'

type Member = Awaited<ReturnType<typeof seedMember>>

async function submit(request: APIRequestContext, token: string, title: string) {
  const deps = await (await request.get('/api/v1/departments', { headers: { Authorization: `Bearer ${token}` } })).json()
  const cs = deps.data.departments.find((d: { code: string }) => d.code === 'CS').id
  const res = await request.post('/api/v1/announcements', {
    data: { title, body: 'Details inside.', category: 'department', audience: [{ department_id: cs, role: 'student' }] },
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(res.ok()).toBeTruthy()
  return (await res.json()).data.id as string
}

test.describe('Approving announcements', () => {
  test.describe.configure({ mode: 'serial' })
  let hod: Member
  let faculty: Member
  let principal: Member
  let reader: Member

  test.beforeAll(async ({ request }) => {
    hod = await seedMember(request, { role: 'hod', fullName: 'Asha Rao', department: 'CS' })
    faculty = await seedMember(request, { role: 'faculty', fullName: 'Meera N', department: 'CS' })
    principal = await seedMember(request, { role: 'principal', fullName: 'S Prakash' })
    reader = await seedMember(request, { role: 'student', fullName: 'Priya Kumar', department: 'CS', batch: 2023 })
  })

  test('faculty submit in the composer, the HOD approves in the queue, and students see it', async ({ browser }) => {
    const desktop = { viewport: { width: 1280, height: 900 } }

    const facultyContext = await browser.newContext(desktop)
    const facultyPage = await facultyContext.newPage()
    await loginViaUI(facultyPage, faculty.email, faculty.password)
    await facultyPage.goto('/mine/new')
    await facultyPage.getByLabel('Title').fill('Lab 2 opens at 9 am')
    await facultyPage.getByLabel('Notice').fill('From Monday, Lab 2 opens at 9 am.')
    await facultyPage.getByRole('button', { name: 'Submit for approval' }).click()
    await facultyPage.waitForURL(/\/mine\/[0-9a-f-]+$/)
    await facultyContext.close()

    const hodContext = await browser.newContext(desktop)
    const hodPage = await hodContext.newPage()
    await loginViaUI(hodPage, hod.email, hod.password)
    await expectHome(hodPage)
    const review = hodPage.getByRole('region', { name: 'Waiting for your review' })
    await review.getByRole('link', { name: /Lab 2 opens at 9 am/ }).click()
    await hodPage.waitForURL(/\/approvals\/[0-9a-f-]+$/)
    await expect(hodPage.getByRole('heading', { level: 2, name: 'Lab 2 opens at 9 am' })).toBeVisible()
    await expect(hodPage.getByText('Approving publishes it to')).toBeVisible()

    await hodPage.getByRole('button', { name: 'Approve', exact: true }).click()
    await expect(hodPage.getByText(/Publish to .* now\?/)).toBeVisible()
    // Focus is on the confirm button, so Enter approves.
    await expect(hodPage.getByRole('button', { name: 'Approve and publish' })).toBeFocused()
    await hodPage.keyboard.press('Enter')
    await expect(hodPage.getByRole('status').filter({ hasText: 'Published.' })).toBeVisible()
    await expect(hodPage.getByRole('heading', { name: 'Nothing waiting for you' })).toBeVisible()
    await hodContext.close()

    const readerContext = await browser.newContext(desktop)
    const readerPage = await readerContext.newPage()
    await loginViaUI(readerPage, reader.email, reader.password)
    await expect(readerPage.getByRole('region', { name: 'Latest notices' }).getByRole('link', { name: /Lab 2 opens at 9 am/ })).toBeVisible()
    await readerPage.getByRole('link', { name: 'Notices' }).first().click()
    await expect(readerPage.getByRole('link', { name: /Lab 2 opens at 9 am/ })).toBeVisible()
    await readerContext.close()
  })

  test('sending back needs a note, and the author sees it', async ({ page, request }) => {
    const facultyToken = await loginViaAPI(request, faculty.email, faculty.password)
    await submit(request, facultyToken, 'Coding club moved to Thursday')

    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, hod.email, hod.password)
    const tabs = page.getByRole('navigation', { name: 'Main' })
    await tabs.getByRole('link', { name: /Approvals/ }).click()
    await page.getByRole('link', { name: /Coding club moved to Thursday/ }).click()

    await page.getByRole('button', { name: 'Send back' }).click()
    await page.getByRole('button', { name: 'Send back' }).click()
    await expect(page.getByText('Add a note so they know what to change.')).toBeVisible()
    await page.getByLabel('What should they change?').fill('Say which room.')
    await page.getByRole('button', { name: 'Send back' }).click()
    await page.waitForURL(/\/approvals$/)
    await expect(page.getByRole('status').filter({ hasText: 'Sent back to Meera N' })).toBeVisible()

    const mine = await (await request.get('/api/v1/announcements/mine?status=attention', { headers: { Authorization: `Bearer ${facultyToken}` } })).json()
    expect(mine.data.map((item: { review_note: string }) => item.review_note)).toContain('Say which room.')
  })

  test('if someone else reviews it first, the queue says so and refreshes', async ({ page, request }) => {
    const facultyToken = await loginViaAPI(request, faculty.email, faculty.password)
    const id = await submit(request, facultyToken, 'Seminar hall booking')

    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, hod.email, hod.password)
    await page.goto(`/approvals/${id}`)
    await expect(page.getByRole('heading', { level: 2, name: 'Seminar hall booking' })).toBeVisible()

    const principalToken = await loginViaAPI(request, principal.email, principal.password)
    const approved = await request.patch(`/api/v1/announcements/${id}/approval`, {
      data: { decision: 'approve' },
      headers: { Authorization: `Bearer ${principalToken}` },
    })
    expect(approved.ok()).toBeTruthy()

    await page.getByRole('button', { name: 'Approve', exact: true }).click()
    await page.getByRole('button', { name: 'Approve and publish' }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Someone else already reviewed "Seminar hall booking"' })).toBeVisible()
    await expect(page.getByRole('link', { name: /Seminar hall booking/ })).toHaveCount(0)
  })

  test('students cannot open the queue', async ({ page }) => {
    await loginViaUI(page, reader.email, reader.password)
    await page.goto('/approvals')
    await page.waitForURL((url) => url.pathname === '/')
    await expectHome(page)
  })
})
