import { test, expect, type Page } from '@playwright/test'
import { bootstrapAdmin, freeUSN, importStudents, loginViaUI, seedMember, setupAdmin, signInWithCode } from '../helpers/auth'
import { getDatabaseURL } from '../helpers/db'

// Not signed in (#127) and fixing a wrong row (#174). CS: other specs
// claim AD, AI, CV, EC and ME for themselves.
const TS = Date.now()

// find scrolls through "Show more" until the row is on screen, as a person
// would with a long list.
async function find(page: Page, text: string) {
  const row = page.getByText(text, { exact: true }).first()
  for (let i = 0; i < 10 && !(await row.isVisible()); i++) {
    const more = page.getByRole('button', { name: 'Show more' })
    if (!(await more.isVisible())) break
    await more.click()
    await page.waitForTimeout(300)
  }
  await expect(row).toBeVisible()
}

test.describe('Not signed in', () => {
  test('an admin fixes a typo in an email, then removes a row', async ({ page, request }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    const token = await setupAdmin(request, getDatabaseURL(), admin)
    const typo = `kavya.${TS}@gmial.com`
    const fixed = `kavya.${TS}@gmail.com`
    const extra = `arjun.${TS}@gmail.com`
    await importStudents(request, token, [
      { email: typo, fullName: `Kavya ${TS}`, usn: await freeUSN('CS', 2025) },
      { email: extra, fullName: `Arjun ${TS}`, usn: await freeUSN('CS', 2025) },
    ])

    await loginViaUI(page, admin.email)
    await page.goto('/admin/not-signed-in')
    await page.getByLabel('Department').selectOption({ label: 'Computer Science and Engineering' })
    await find(page, `Kavya ${TS}`)

    await page.getByRole('button', { name: `More for Kavya ${TS}` }).click()
    await page.getByRole('button', { name: 'Fix email' }).click()
    const field = page.getByRole('textbox', { name: `Kavya ${TS}'s email` })
    await expect(field).toHaveValue(typo)
    await field.fill(fixed)
    await page.getByRole('button', { name: 'Save email' }).click()
    await expect(page.getByRole('status')).toHaveText(`Saved. Kavya ${TS} now signs in with ${fixed}.`)
    await expect(page.getByRole('cell', { name: fixed })).toBeVisible()

    await page.getByRole('button', { name: `More for Arjun ${TS}` }).click()
    await page.getByRole('button', { name: 'Remove from the list' }).click()
    await expect(page.getByText(`Remove Arjun ${TS} from the list?`)).toBeVisible()
    await page.getByRole('button', { name: 'Remove', exact: true }).click()
    await expect(page.getByRole('status')).toContainText(`Removed Arjun ${TS}.`)
    await expect(page.getByRole('cell', { name: extra })).toHaveCount(0)

    // Kavya signs in with the fixed email.
    const kavya = await page.context().browser()!.newPage()
    await signInWithCode(kavya, fixed)
    await expect(kavya.getByRole('button', { name: "Yes, that's me" })).toBeVisible()
  })

  test("an HOD opens their department's list from Home", async ({ page, request }) => {
    const hod = await seedMember(request, { role: 'hod', fullName: 'Meera Iyer', department: 'CS' })
    const admin = await bootstrapAdmin(getDatabaseURL())
    const token = await setupAdmin(request, getDatabaseURL(), admin)
    await importStudents(request, token, [{ email: `ravi.${TS}@gmail.com`, fullName: `Ravi ${TS}`, usn: await freeUSN('CS', 2025) }])

    await loginViaUI(page, hod.email)
    await page.getByRole('link', { name: /See who/ }).click()
    await expect(page.getByRole('heading', { name: /Not signed in yet/ })).toBeVisible()
    await expect(page.getByLabel('Department')).toHaveCount(0)
    await find(page, `Ravi ${TS}`)
  })

  test("an admin fixes a reported row's email in Access requests", async ({ page, request, browser }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    const token = await setupAdmin(request, getDatabaseURL(), admin)
    const wrong = `rohan.${TS}@outlook.com`
    const right = `rohan.shetty.${TS}@gmail.com`
    await importStudents(request, token, [{ email: wrong, fullName: `Rohan ${TS}`, usn: await freeUSN('CS', 2025) }])

    const stranger = await browser.newPage()
    await signInWithCode(stranger, wrong)
    await stranger.getByRole('button', { name: 'Not you? Report it' }).click()
    await stranger.getByRole('button', { name: 'Report and sign out' }).click()
    await stranger.waitForURL('**/login')

    await loginViaUI(page, admin.email)
    await page.goto('/admin/requests')
    await page.getByText(`Rohan ${TS}`).first().click()
    await page.getByRole('button', { name: 'Fix email' }).click()
    await expect(page.getByRole('button', { name: 'Approve' })).toHaveCount(0)
    await page.getByRole('textbox', { name: `Rohan ${TS}'s email` }).fill(right)
    await page.getByRole('button', { name: 'Save email' }).click()
    await expect(page.getByText(`Saved. Rohan ${TS} now signs in with ${right}.`)).toBeVisible()
    await expect(page.getByText(`Rohan ${TS}`, { exact: true })).toHaveCount(0)
  })
})
