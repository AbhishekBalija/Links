import { test, expect } from '@playwright/test'
import { bootstrapAdmin, loginViaUI, seedMember, setupAdmin, signInWithCode } from '../helpers/auth'
import { getDatabaseURL, getSchemaClient } from '../helpers/db'

// Add staff (#189): an admin adds staff, who then sign in with that email;
// an HOD adds faculty to their own department. CS, since proposing.spec
// keeps EC to itself.
const TS = Date.now()

async function emailOf(userId: string): Promise<string> {
  const client = await getSchemaClient()
  try {
    return (await client.query('SELECT email FROM users WHERE id = $1', [userId])).rows[0].email
  } finally {
    await client.end()
  }
}

test.describe('Add staff', () => {
  test('an admin adds a faculty member, who then signs in with that email', async ({ page, request, browser }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    await setupAdmin(request, getDatabaseURL(), admin)
    const email = `kiran.${TS}@college.edu`

    await loginViaUI(page, admin.email)
    await page.goto('/admin/staff')
    await expect(page.getByRole('link', { name: 'Add staff' })).toHaveAttribute('aria-current', 'page')

    await page.getByLabel('Full name').fill('Prof. Kiran Hegde')
    await page.getByLabel('Email').fill(email)
    await page.getByLabel('Role').selectOption({ label: 'Faculty' })
    await page.getByLabel('Department').selectOption({ label: 'Computer Science and Engineering' })
    await page.getByRole('button', { name: 'Add staff member' }).click()

    await expect(page.getByRole('status')).toContainText('Prof. Kiran Hegde was added as Faculty, Computer Science and Engineering.')
    await expect(page.getByLabel('Full name')).toHaveValue('')
    await expect(page.getByLabel('Full name')).toBeFocused()
    await expect(page.getByLabel('Role')).toHaveValue('faculty')

    const theirs = await (await browser.newContext()).newPage()
    await signInWithCode(theirs, email)
    await expect(theirs.getByText('Faculty, Computer Science and Engineering')).toBeVisible()
  })

  test("an admin is told when an email is a current student's", async ({ page, request }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    await setupAdmin(request, getDatabaseURL(), admin)
    const student = await seedMember(request, { role: 'student', fullName: 'Asha Rao', department: 'CS', batch: 2023 })
    const email = await emailOf(student.userId)

    await loginViaUI(page, admin.email)
    await page.goto('/admin/staff')
    await page.getByLabel('Full name').fill('Asha Rao')
    await page.getByLabel('Email').fill(email)
    await page.getByLabel('Role').selectOption({ label: 'Faculty' })
    await page.getByLabel('Department').selectOption({ label: 'Computer Science and Engineering' })
    await page.getByRole('button', { name: 'Add staff member' }).click()

    await expect(page.getByLabel('Email')).toHaveAttribute('aria-invalid', 'true')
    await expect(page.getByText(`${email} is a current student's email. Staff need their own email.`)).toBeVisible()
  })

  test('Home sends the admin to add an HOD with the department chosen', async ({ page, request }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    await setupAdmin(request, getDatabaseURL(), admin)

    await loginViaUI(page, admin.email)
    await page.goto('/admin/staff?role=hod&department=ME')
    await expect(page.getByLabel('Role')).toHaveValue('hod')
    await expect(page.getByLabel('Department')).toHaveValue(/.+/)
    await expect(page.getByLabel('Department').locator('option:checked')).toHaveText('Mechanical Engineering')
  })

  test('an HOD adds faculty to their own department from Home', async ({ page, request }) => {
    const hod = await seedMember(request, { role: 'hod', fullName: 'Meera Iyer', department: 'CS' })
    const email = `ravi.${TS}@college.edu`

    await loginViaUI(page, hod.email)
    await page.getByRole('link', { name: 'Add faculty →' }).click()
    await expect(page.getByRole('heading', { name: 'Add faculty', level: 1 })).toBeVisible()
    await expect(page.getByText('Faculty, Computer Science and Engineering')).toBeVisible()

    await page.getByLabel('Full name').fill('Ravi Kumar')
    await page.getByLabel('Email').fill(email)
    await page.getByRole('button', { name: 'Add faculty member' }).click()
    await expect(page.getByRole('status')).toContainText('Ravi Kumar was added as Faculty, Computer Science and Engineering.')
  })
})
