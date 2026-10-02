import { test, expect } from '@playwright/test'
import { getDatabaseURL } from '../helpers/db'
import {
  adminApproveUser,
  bootstrapAdmin,
  cleanupTestUsers,
  freeUSN,
  loginViaAPI,
  loginViaUI,
  seedMember,
  sendAccessRequest,
  setupAdmin,
  signInWithCode,
} from '../helpers/auth'

// Access requests (#124): an HOD decides them in their Approval queue, an
// admin in the Admin workspace.
const TS = Date.now()
const APPROVED = `e2e-approve-${TS}@test.com`
const REJECTED = `e2e-reject-${TS}@test.com`
const RACED = `e2e-race-${TS}@test.com`

test.describe('Access requests', () => {
  let dbURL: string
  let adminEmail: string
  const emails = [APPROVED, REJECTED, RACED]

  test.beforeAll(async ({ request }) => {
    dbURL = getDatabaseURL()
    const admin = await bootstrapAdmin(dbURL)
    await setupAdmin(request, dbURL, admin)
    adminEmail = admin.email
    emails.push(admin.email)
  })

  test.afterAll(async () => {
    await cleanupTestUsers(dbURL, emails)
  })

  test('an HOD approves a request in the Approval queue and the student can sign in', async ({ page, request }) => {
    const hod = await seedMember(request, { role: 'hod', fullName: 'Asha Rao', department: 'CS' })
    emails.push(hod.email)
    await sendAccessRequest(request, APPROVED, 'Kiran Shetty', await freeUSN('CS', 2024))

    await loginViaUI(page, hod.email)
    await page.goto('/approvals')
    await page.getByRole('navigation', { name: "What's waiting" }).getByRole('link', { name: /Access requests/ }).click()
    await page.waitForURL('**/approvals/access**')
    await page.getByRole('link', { name: /Kiran Shetty/ }).click()
    await expect(page.getByRole('heading', { name: 'Kiran Shetty' })).toBeVisible()
    await expect(page.getByText('Not on the Computer Science and Engineering class list')).toBeVisible()

    await page.getByRole('button', { name: 'Approve' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Approve' }).click()
    await expect(page.getByRole('status')).toContainText('Kiran Shetty can sign in now')

    const student = await page.context().browser()!.newPage()
    await signInWithCode(student, APPROVED)
    await student.waitForURL('**/welcome')
    await student.close()
  })

  test('an admin rejects a request with a note', async ({ page, request }) => {
    await sendAccessRequest(request, REJECTED, 'Ravi Kumar', await freeUSN('EC', 2024))

    await loginViaUI(page, adminEmail)
    await page.getByRole('navigation', { name: 'Main' }).first().getByRole('link', { name: 'Admin' }).click()
    await page.waitForURL('**/admin/requests**')
    await page.getByRole('link', { name: /Ravi Kumar/ }).click()
    await expect(page.getByText('has no HOD, so the request comes to admins')).toBeVisible()

    await page.getByRole('button', { name: 'Reject', exact: true }).click()
    await page.getByRole('button', { name: 'Reject request' }).click()
    await expect(page.getByText("Write a short reason. It's kept with the decision.")).toBeVisible()
    await page.getByRole('textbox').fill('USN not in our records for 2024.')
    await page.getByRole('button', { name: 'Reject request' }).click()
    await expect(page.getByRole('status')).toContainText("Ravi Kumar's request was rejected")
  })

  test('a reviewer who decides too late is told someone else already did', async ({ page, request }) => {
    const id = await sendAccessRequest(request, RACED, 'Meera Nair', await freeUSN('EC', 2024))

    await loginViaUI(page, adminEmail)
    await page.goto(`/admin/requests/${id}`)
    await expect(page.getByRole('heading', { name: 'Meera Nair' })).toBeVisible()
    // Another admin approves it while this page is open.
    await adminApproveUser(request, await loginViaAPI(request, adminEmail), id)

    await page.getByRole('button', { name: 'Approve' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Approve' }).click()
    await expect(page.getByRole('status')).toContainText("Someone else already decided Meera Nair's request")
  })
})

test('when the list fails to load, the page says so and stops asking', async ({ page, request }) => {
  const hod = await seedMember(request, { role: 'hod', fullName: 'Leela M', department: 'CS' })
  await loginViaUI(page, hod.email)
  let asked = 0
  await page.route('**/api/v1/admin/users/review-queue', (route) => {
    asked++
    return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":{"code":"INTERNAL_ERROR","message":"down"}}' })
  })
  await page.goto('/approvals/access')
  await expect(page.getByRole('alert')).toContainText('Access requests could not be loaded.', { timeout: 15_000 })
  const settled = asked
  await page.waitForTimeout(4000)
  expect(asked, 'the page kept re-requesting a failed list').toBe(settled)
  await cleanupTestUsers(getDatabaseURL(), [hod.email])
})
