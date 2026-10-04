import { test, expect, type APIRequestContext } from '@playwright/test'
import { bootstrapAdmin, loginViaAPI, loginViaUI, seedMember, setupAdmin } from '../helpers/auth'
import { getSchemaClient } from '../helpers/db'

// After a handover (ADR 0028) the new Organiser runs the event and finds it
// in My posts, and the person whose role ended no longer does (#205).
// CS: other specs keep AD, AI, CV, EC and ME to themselves.
const TS = Date.now()
const auth = (token: string) => ({ Authorization: `Bearer ${token}` })

async function call(request: APIRequestContext, method: 'post' | 'patch' | 'get' | 'delete', path: string, token: string, data?: unknown) {
  const res = await request[method](path, { data, headers: auth(token) })
  if (!res.ok()) throw new Error(`${method} ${path} failed (${res.status()}): ${await res.text()}`)
  return (await res.json()).data
}

async function visible(userIds: string[]) {
  const client = await getSchemaClient()
  try {
    await client.query(`UPDATE users SET is_verified = true WHERE id = ANY($1)`, [userIds])
    return (await client.query('SELECT id FROM departments WHERE code = $1', ['CS'])).rows[0].id as string
  } finally {
    await client.end()
  }
}

test('a handed-over event is run by its new Organiser', async ({ page, request }) => {
  const proposer = await seedMember(request, { role: 'faculty', fullName: `Nisha ${TS}`, department: 'CS' })
  const colleague = await seedMember(request, { role: 'faculty', fullName: `Ravi ${TS}`, department: 'CS' })
  const hod = await seedMember(request, { role: 'hod', fullName: 'Meera Iyer', department: 'CS' })
  const principal = await seedMember(request, { role: 'principal', fullName: 'Principal Iyer' })
  const admin = await bootstrapAdmin('')
  const adminToken = await setupAdmin(request, '', admin)
  const cs = await visible([proposer.userId, colleague.userId, hod.userId])

  const starts = new Date(Date.now() + 5 * 24 * 3600 * 1000)
  starts.setHours(11, 0, 0, 0)
  const created = await call(request, 'post', '/api/v1/events', await loginViaAPI(request, proposer.email), {
    title: `Robotics meetup ${TS}`,
    description: 'Bring your bots.',
    event_type: 'workshop',
    department_id: cs,
    location: 'CS Lab 2',
    starts_at: starts.toISOString(),
    ends_at: new Date(starts.getTime() + 2 * 3600 * 1000).toISOString(),
    audience: [{ department_id: cs }],
  })
  await call(request, 'patch', `/api/v1/events/${created.id}/hod-review`, await loginViaAPI(request, hod.email), { decision: 'approve', note: '' })
  await call(request, 'patch', `/api/v1/events/${created.id}/final-approval`, await loginViaAPI(request, principal.email), { decision: 'approve', note: '' })

  // The proposer's faculty role ends; the event goes to their colleague.
  const roles = await call(request, 'get', `/api/v1/admin/users/${proposer.userId}/roles`, adminToken)
  const faculty = (roles.roles ?? roles).find((r: { role: string }) => r.role === 'faculty')
  await call(request, 'delete', `/api/v1/admin/users/${proposer.userId}/roles/${faculty.id}?organiser_id=${colleague.userId}`, adminToken)

  await loginViaUI(page, colleague.email)
  await page.goto('/mine')
  await page.getByRole('link', { name: new RegExp(`Robotics meetup ${TS}`) }).first().click()
  await page.getByRole('link', { name: 'Open event page' }).click()
  await expect(page.getByText(`Ravi ${TS}`, { exact: true })).toBeVisible()
  await expect(page.getByRole('complementary', { name: "Who's coming" })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Edit details' })).toBeVisible()
})
