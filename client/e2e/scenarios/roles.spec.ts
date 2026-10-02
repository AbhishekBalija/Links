import { test, expect } from '@playwright/test'
import { loginViaUI, seedMember } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// Roles on a profile (#125): an HOD appoints and removes student
// coordinators (ADR 0027); the principal appoints senior staff.

// visible makes seeded members show up as an imported or approved account
// would, and returns their usernames for their profile pages.
async function visible(userIds: string[]): Promise<string[]> {
  const client = await getSchemaClient()
  try {
    await client.query(`UPDATE users SET is_verified = true WHERE id = ANY($1)`, [userIds])
    const usernames: string[] = []
    for (const id of userIds) {
      usernames.push((await client.query('SELECT username FROM profiles WHERE user_id = $1', [id])).rows[0].username)
    }
    return usernames
  } finally {
    await client.end()
  }
}

test.describe('Roles on a profile', () => {
  test('an HOD makes a student a coordinator, then ends the role', async ({ page, request }) => {
    // AI is used by no other spec, so this HOD is its only one.
    const hod = await seedMember(request, { role: 'hod', fullName: 'Meera Iyer', department: 'AI' })
    const student = await seedMember(request, { role: 'student', fullName: 'Rohan Shetty', department: 'AI', batch: 2025 })
    const [username] = await visible([student.userId, hod.userId])

    await loginViaUI(page, hod.email)
    await page.goto(`/people/${username}`)
    const roles = page.getByRole('region', { name: 'Roles' })
    await expect(roles.getByText('Student', { exact: true })).toBeVisible()

    await roles.getByRole('button', { name: 'Make student coordinator' }).click()
    await expect(roles.getByText('Make Rohan a student coordinator')).toBeVisible()
    await roles.getByRole('button', { name: 'Make coordinator' }).click()
    await expect(roles.getByRole('status')).toHaveText('Rohan is a student coordinator now.')
    await expect(roles.getByText('Student coordinator', { exact: true })).toBeVisible()
    // One coordinator role at a time, so the button goes away.
    await expect(roles.getByRole('button', { name: 'Make student coordinator' })).toHaveCount(0)

    await roles.getByRole('button', { name: 'More for Student coordinator' }).click()
    await page.getByRole('menuitem', { name: 'End role' }).click()
    const confirm = page.getByRole('alertdialog', { name: "End Rohan's Student coordinator role today?" })
    await expect(confirm.getByText("Nothing of Rohan's is waiting for approval or coming up.")).toBeVisible()
    await expect(confirm.getByText('They stay a student and see their old posts read-only.')).toBeVisible()
    await confirm.getByRole('button', { name: 'End role' }).click()

    await expect(roles.getByRole('status')).toHaveText("Rohan's Student coordinator role has ended.")
    await expect(roles.getByText('Ended', { exact: true })).toBeVisible()
    await expect(roles.getByRole('button', { name: 'Make student coordinator' })).toBeVisible()
  })

  test('the principal grants a faculty role, and never a coordinator one', async ({ page, request }) => {
    const principal = await seedMember(request, { role: 'principal', fullName: 'Suresh Kumar' })
    const teacher = await seedMember(request, { role: 'placement_officer', fullName: 'Kiran Hegde' })
    const [username] = await visible([teacher.userId, principal.userId])

    await loginViaUI(page, principal.email)
    await page.goto(`/people/${username}`)
    const roles = page.getByRole('region', { name: 'Roles' })
    await roles.getByRole('button', { name: 'Grant a role' }).click()

    const role = roles.getByLabel('Role', { exact: true })
    await expect(role.locator('option')).toHaveText(['Faculty', 'HOD', 'Placement officer'])
    await role.selectOption({ label: 'Faculty' })
    await roles.getByRole('button', { name: 'Grant role' }).click()
    await expect(roles.getByText('A Faculty role needs a department.')).toBeVisible()

    await roles.getByLabel('Department').selectOption({ label: 'Computer Science and Engineering (AI and ML)' })
    await roles.getByRole('button', { name: 'Grant role' }).click()
    await expect(roles.getByRole('status')).toHaveText('Kiran has the Faculty role now.')
    await expect(roles.getByText('Faculty', { exact: true })).toBeVisible()
  })
})
