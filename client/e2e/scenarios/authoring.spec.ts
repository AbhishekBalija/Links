import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginViaAPI, loginViaUI, seedMember, expectHome } from '../helpers/auth'

type Member = Awaited<ReturnType<typeof seedMember>>

async function feedTitles(request: APIRequestContext, token: string): Promise<string[]> {
  const res = await request.get('/api/v1/announcements?limit=50', { headers: { Authorization: `Bearer ${token}` } })
  const body = await res.json()
  return body.data.map((item: { title: string }) => item.title)
}

async function mineByTitle(request: APIRequestContext, token: string, title: string): Promise<string> {
  const res = await request.get('/api/v1/announcements/mine?limit=50', { headers: { Authorization: `Bearer ${token}` } })
  const body = await res.json()
  return body.data.find((item: { title: string; id: string }) => item.title === title).id
}

test.describe('Writing announcements', () => {
  test.describe.configure({ mode: 'serial' })
  let hod: Member
  let faculty: Member
  let reader: Member

  test.beforeAll(async ({ request }) => {
    hod = await seedMember(request, { role: 'hod', fullName: 'Asha Rao', department: 'CS' })
    faculty = await seedMember(request, { role: 'faculty', fullName: 'Meera N', department: 'CS' })
    reader = await seedMember(request, { role: 'student', fullName: 'Priya Kumar', department: 'CS', batch: 2023 })
  })

  test('an HOD posts to their own department and it publishes straight away', async ({ page, request }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, hod.email, hod.password)
    await expectHome(page)

    await page.getByRole('link', { name: 'Notices' }).first().click()
    await page.getByRole('link', { name: 'New announcement' }).click()
    await page.waitForURL('**/mine/new')

    await page.getByLabel('Title').fill('CS lab 3 closed on Monday')
    await page.getByLabel('Notice').fill('Use lab 2 instead.')
    // The usual case is one tap: the author's department's students.
    await expect(page.getByRole('radio', { name: /CS students/ })).toBeChecked()
    await expect(page.getByText('Publishes now.')).toBeVisible()

    await page.getByRole('button', { name: 'Publish', exact: true }).click()
    await page.waitForURL(/\/mine\/[0-9a-f-]+$/)
    await expect(page.getByRole('heading', { level: 1, name: 'CS lab 3 closed on Monday' })).toBeVisible()
    await expect(page.getByRole('complementary').getByText('Live', { exact: true })).toBeVisible()

    const readerToken = await loginViaAPI(request, reader.email, reader.password)
    expect(await feedTitles(request, readerToken)).toContain('CS lab 3 closed on Monday')
  })

  test('faculty submit, get it sent back, fix it and resubmit', async ({ page, request }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, faculty.email, faculty.password)
    await page.getByRole('link', { name: 'My announcements' }).click()
    await expect(page.getByRole('heading', { name: "You haven't posted anything yet" })).toBeVisible()

    await page.getByRole('link', { name: 'New announcement' }).click()
    await page.getByLabel('Title').fill('Lab 2 timings change')
    await page.getByLabel('Notice').fill('Lab 2 opens at 9 am from Monday.')
    await expect(page.getByText(/Goes to the CS HOD for approval/)).toBeVisible()
    await page.getByRole('button', { name: 'Submit for approval' }).click()

    await page.waitForURL(/\/mine\/[0-9a-f-]+$/)
    await expect(page.getByText('You can edit it once they decide.')).toBeVisible()
    await expect(page.getByRole('link', { name: 'Edit', exact: true })).toHaveCount(0)

    // The HOD sends it back with a note (the approval queue is #42).
    const hodToken = await loginViaAPI(request, hod.email, hod.password)
    const facultyToken = await loginViaAPI(request, faculty.email, faculty.password)
    const id = await mineByTitle(request, facultyToken, 'Lab 2 timings change')
    const rejected = await request.patch(`/api/v1/announcements/${id}/approval`, {
      data: { decision: 'reject', note: 'Add the Thursday timings too.' },
      headers: { Authorization: `Bearer ${hodToken}` },
    })
    expect(rejected.ok()).toBeTruthy()

    await page.goto('/mine')
    const card = page.getByRole('region', { name: /Sent back to you/ })
    await expect(card.getByText('Add the Thursday timings too.')).toBeVisible()
    await card.getByRole('link', { name: 'Edit and resubmit' }).click()

    await expect(page.getByRole('note')).toContainText('Add the Thursday timings too.')
    await page.getByLabel('Notice').fill('Lab 2 opens at 9 am from Monday, and at 10 am on Thursdays.')
    await page.getByRole('button', { name: 'Submit for approval' }).click()
    await page.waitForURL(/\/mine\/[0-9a-f-]+$/)
    await expect(page.getByText('You can edit it once they decide.')).toBeVisible()

    await page.goto('/mine')
    await expect(page.getByRole('region', { name: /Sent back to you/ })).toHaveCount(0)
  })

  test('a missing title is caught, and the text already written stays', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, faculty.email, faculty.password)
    await page.goto('/mine/new')
    await page.getByLabel('Notice').fill('Bring your ID cards.')
    await page.getByRole('button', { name: 'Submit for approval' }).click()

    await expect(page.getByText('Add a title so readers know what this is about.')).toBeVisible()
    await expect(page.getByLabel('Title')).toBeFocused()
    await expect(page.getByLabel('Notice')).toHaveValue('Bring your ID cards.')
  })

  test('leaving with unsaved writing offers to keep it as a draft', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await loginViaUI(page, faculty.email, faculty.password)
    await page.goto('/mine/new')
    await page.getByLabel('Title').fill('Seminar on embedded systems')
    await page.getByLabel('Notice').fill('Details to follow.')

    await page.getByRole('link', { name: 'My announcements' }).first().click()
    const dialog = page.getByRole('dialog', { name: 'Keep this as a draft?' })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: 'Keep editing' }).click()
    await expect(page.getByLabel('Title')).toHaveValue('Seminar on embedded systems')

    await page.getByRole('link', { name: 'My announcements' }).first().click()
    await page.getByRole('dialog', { name: 'Keep this as a draft?' }).getByRole('button', { name: 'Save draft' }).click()
    // Saving carries on to where they were going, with the draft kept.
    await page.waitForURL(/\/mine$/)
    await expect(page.getByRole('link', { name: /Seminar on embedded systems.*saved/ })).toBeVisible()
  })

  test('withdrawing asks first, then takes the notice down', async ({ page, request }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginViaUI(page, hod.email, hod.password)
    await page.goto('/mine?status=live')
    await page.getByRole('link', { name: /CS lab 3 closed on Monday/ }).click()

    await page.getByRole('button', { name: 'Withdraw' }).click()
    await expect(page.getByText('Take this down for everyone?')).toBeVisible()
    await page.getByRole('button', { name: 'Keep it' }).click()
    await expect(page.getByText('Take this down for everyone?')).toHaveCount(0)

    await page.getByRole('button', { name: 'Withdraw' }).click()
    await page.getByRole('button', { name: 'Withdraw' }).click()
    await page.waitForURL('**/mine?status=ended')
    await expect(page.getByText('Withdrawn', { exact: true })).toBeVisible()

    const readerToken = await loginViaAPI(request, reader.email, reader.password)
    expect(await feedTitles(request, readerToken)).not.toContain('CS lab 3 closed on Monday')
  })
})
