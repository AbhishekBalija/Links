import { test, expect } from '@playwright/test'
import { bootstrapAdmin, freeUSN, loginViaUI, seedMember, setupAdmin } from '../helpers/auth'
import { getDatabaseURL } from '../helpers/db'

// Import students (#126): check every row first, nothing saved; then import.
const TS = Date.now()

function classList(rows: Array<[string, string, string]>) {
  return Buffer.from(['email,full_name,usn', ...rows.map((r) => r.join(','))].join('\n') + '\n')
}

test.describe('Import students', () => {
  test('an admin checks a class list for two departments, then imports it', async ({ page, request }) => {
    const admin = await bootstrapAdmin(getDatabaseURL())
    await setupAdmin(request, getDatabaseURL(), admin)
    const cs = await freeUSN('CS', 2025)
    const ec = await freeUSN('EC', 2025)

    await loginViaUI(page, admin.email)
    await page.goto('/admin/import')
    await expect(page.getByRole('heading', { name: 'Upload a class list' })).toBeVisible()

    await page.locator('input[type=file]').first().setInputFiles({
      name: 'batch-2025.csv',
      mimeType: 'text/csv',
      buffer: classList([
        [`kavya.${TS}@gmail.com`, 'Kavya Rao', cs],
        [`arjun.${TS}@gmail.com`, 'Arjun Nayak', ec],
        [`rahul.${TS}@gmail.com`, 'Rahul P', '4MN25CS02'],
      ]),
    })

    await expect(page.getByRole('heading', { name: 'Check before importing' })).toBeVisible()
    await expect(page.getByRole('cell', { name: 'Kavya Rao' })).toBeVisible()
    await expect(page.getByText("Won't be added: the USN can't be read")).toBeVisible()
    await expect(page.getByText('Nothing is saved until you import.')).toBeVisible()

    await page.getByRole('button', { name: 'Import 2 students' }).click()
    await expect(page.getByText('The 2 can sign in now.')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Not added' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Download the row' })).toBeVisible()
  })

  test("an HOD's rows for other departments are flagged and not added", async ({ page, request }) => {
    // CV is used by no other spec, so this HOD is its only one.
    const hod = await seedMember(request, { role: 'hod', fullName: 'Leela Gowda', department: 'CV' })
    const cv = await freeUSN('CV', 2024)
    const me = await freeUSN('ME', 2024)

    await loginViaUI(page, hod.email)
    await page.goto('/import')
    await expect(page.getByText(/Only .* USNs/)).toBeVisible()

    await page.locator('input[type=file]').first().setInputFiles({
      name: 'cv-2024.csv',
      mimeType: 'text/csv',
      buffer: classList([
        [`nisha.${TS}@gmail.com`, 'Nisha K', cv],
        [`varun.${TS}@gmail.com`, 'Varun M', me],
      ]),
    })

    await expect(page.getByText(/so rows for other departments won't be added/)).toBeVisible()
    await expect(page.getByText(/Won't be added: mechanical engineering, not/i)).toBeVisible()
    await page.getByRole('button', { name: 'Import 1 student' }).click()
    await expect(page.getByText(/The 1 can sign in now\. Send the row for other departments to their HOD\./)).toBeVisible()
  })
})
