import { expect, type Page, type APIRequestContext } from '@playwright/test'
import { getSchemaClient } from './db'

// ── Admin bootstrap (seeded directly — not the thing under test) ──
export async function bootstrapAdmin(_dbURL: string) {
  void _dbURL
  const { v4: uuidv4 } = await import('uuid')
  const { randomBytes, createHash } = await import('crypto')

  const adminEmail = `admin-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@test.com`
  const adminPass = 'AdminPass123'
  const client = await getSchemaClient()
  try {
    const userId = uuidv4()
    const now = new Date().toISOString()

    // Create user as pending with placeholder hash (overwritten by activate endpoint)
    const placeholderHash = createHash('sha256').update(adminPass).digest('hex')
    await client.query(
      `INSERT INTO users (id, email, password_hash, status, is_verified, created_at, updated_at)
       VALUES ($1, $2, $3, 'pending', false, $4, $4)`,
      [userId, adminEmail, placeholderHash, now]
    )
    await client.query(
      `INSERT INTO profiles (user_id, username, full_name, created_at, updated_at)
       VALUES ($1, $2, $3, $4, $4)`,
      [userId, `admin_${uuidv4().slice(0, 8)}`, 'E2E Admin', now]
    )
    // Activation token (for setting the real password hash via /activate)
    const tokenRaw = randomBytes(32).toString('base64url')
    const tokenHash = createHash('sha256').update(tokenRaw).digest('hex')
    await client.query(
      `INSERT INTO account_activation_tokens (id, user_id, token_hash, purpose, expires_at, created_at)
       VALUES ($1, $2, $3, 'activate', $4, $5)`,
      [uuidv4(), userId, tokenHash, new Date(Date.now() + 7 * 86400000).toISOString(), now]
    )

    return { email: adminEmail, password: adminPass, userId, activationToken: tokenRaw }
  } finally {
    await client.end()
  }
}

// ── Activate and assign role to a bootstrapped admin (used in beforeAll) ──
export async function setupAdmin(
  apiContext: APIRequestContext,
  _dbURL: string,
  admin: { email: string; password: string; userId: string; activationToken: string }
): Promise<string> {
  await activateUserViaAPI(apiContext, admin.activationToken, admin.password)
  const { v4: uuidv4 } = await import('uuid')
  const dbClient = await getSchemaClient()
  try {
    await dbClient.query(`UPDATE users SET status = 'active' WHERE id = $1`, [admin.userId])
    await dbClient.query(
      `INSERT INTO role_assignments (id, user_id, role, scope_type, starts_at, created_at)
       VALUES ($1, $2, 'admin', 'global', NOW(), NOW())`,
      [uuidv4(), admin.userId]
    )
  } finally {
    await dbClient.end()
  }
  return await loginViaAPI(apiContext, admin.email, admin.password)
}

// ── UI helpers ──
// testSignIn signs a seeded member in through the test-only endpoint
// (ENABLE_TEST_SIGN_IN, local only), so the suite doesn't depend on how
// people really sign in. Specs about the sign-in screens drive those instead.
async function testSignIn(apiContext: APIRequestContext, email: string): Promise<string> {
  const res = await apiContext.post('/api/v1/test/sign-in', { data: { email } })
  if (!res.ok()) {
    throw new Error(`Test sign-in failed (${res.status()}): ${await res.text()}`)
  }
  return (await res.json()).data.access_token
}

// loginViaUI signs the page's browser in and opens the app. The app restores
// the session from the refresh cookie, and that refresh swaps the cookie for a
// new one, so this waits for the refresh to answer: moving on before the new
// cookie lands would leave the browser holding a retired one.
export async function loginViaUI(page: Page, email: string) {
  await testSignIn(page.request, email)
  const refreshed = page.waitForResponse((res) => res.url().includes('/api/v1/auth/refresh'))
  await page.goto('/')
  const response = await refreshed
  expect(response.ok(), 'the app could not restore the session').toBe(true)
  await page.waitForURL((url) => !url.pathname.includes('/login'))
}

// signInWithCode goes through the sign-in screens with a real email code:
// the email, then the code the server emailed (read back through the
// test-only endpoint), then Sign in. It stops there, so each spec checks
// where the sign-in lands.
export async function signInWithCode(page: Page, email: string) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByRole('button', { name: 'Email me a code' }).click()
  await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()
  await typeCode(page, await lastCode(page.request, email))
}

export async function typeCode(page: Page, code: string) {
  await page.getByLabel('6-digit code').fill(code)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
}

export async function lastCode(apiContext: APIRequestContext, email: string): Promise<string> {
  const res = await apiContext.get(`/api/v1/test/sign-in-code?email=${encodeURIComponent(email)}`)
  if (!res.ok()) throw new Error(`No code was sent to ${email} (${res.status()})`)
  return (await res.json()).data.code
}

// importStudents adds class list rows the way an admin does.
export async function importStudents(apiContext: APIRequestContext, adminToken: string, rows: Array<{ email: string; fullName: string; usn: string }>) {
  const csv = ['email,full_name,usn', ...rows.map((r) => `${r.email},${r.fullName},${r.usn}`)].join('\n')
  const res = await apiContext.post('/api/v1/admin/users/import', {
    headers: { Authorization: `Bearer ${adminToken}` },
    multipart: { file: { name: 'class-list.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) } },
  })
  if (!res.ok()) throw new Error(`Import failed (${res.status()}): ${await res.text()}`)
  const body = await res.json()
  if (body.data.failed > 0) throw new Error(`Import rows failed: ${JSON.stringify(body.data.rows)}`)
}

// freeUSN picks a USN no one has yet. USNs are unique and other specs seed
// students too, so it checks rather than trusting a random roll number.
export async function freeUSN(department: string, batch: number): Promise<string> {
  const client = await getSchemaClient()
  try {
    for (let attempt = 0; attempt < 50; attempt++) {
      const candidate = `4MN${String(batch).slice(2)}${department}${String(Math.floor(Math.random() * 900) + 100)}`
      const taken = await client.query('SELECT 1 FROM student_identities WHERE lower(usn) = lower($1)', [candidate])
      if (taken.rowCount === 0) return candidate
    }
  } finally {
    await client.end()
  }
  throw new Error('no free USN found')
}

// ── DB extraction helpers ──
export async function getUserIdByEmail(_dbURL: string, email: string): Promise<string> {
  const client = await getSchemaClient()
  try {
    const result = await client.query('SELECT id FROM users WHERE email = $1', [email])
    return result.rows[0]?.id
  } finally {
    await client.end()
  }
}

// ── API helpers (for endpoints without a UI yet) ──
export async function adminApproveUser(apiContext: APIRequestContext, adminToken: string, userId: string) {
  const res = await apiContext.patch(`/api/v1/admin/users/${userId}/verify`, {
    data: {},
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!res.ok()) {
    const body = await res.text()
    throw new Error(`Approval failed (${res.status()}): ${body}`)
  }
}

export async function activateUserViaAPI(apiContext: APIRequestContext, token: string, password: string) {
  const res = await apiContext.post('/api/v1/auth/activate', {
    data: { token, password },
  })
  if (!res.ok()) {
    const body = await res.text()
    throw new Error(`Activation failed (${res.status()}): ${body}`)
  }
}

// loginViaAPI returns an access token for API calls made outside the page.
export async function loginViaAPI(apiContext: APIRequestContext, email: string): Promise<string> {
  return testSignIn(apiContext, email)
}

// ── Cleanup ──
export async function cleanupTestUsers(_dbURL: string, emails: string[]) {
  const client = await getSchemaClient()
  try {
	for (const email of emails) {
		await client.query(`DELETE FROM audit_logs WHERE actor_id IN (SELECT id FROM users WHERE email = $1) OR resource_id IN (SELECT id FROM users WHERE email = $1)`, [email])
		await client.query(`DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
      await client.query(`DELETE FROM role_assignments WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
      await client.query(`DELETE FROM account_activation_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
      await client.query(`DELETE FROM student_identities WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
      await client.query(`DELETE FROM profiles WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
      await client.query(`DELETE FROM users WHERE email = $1`, [email])
    }
  } finally {
    await client.end()
  }
}

// ── Home ──
// Home greets the user by time of day: "Good morning, Priya".
export async function expectHome(page: Page) {
  await expect(page.getByRole('heading', { level: 1, name: /^Good (morning|afternoon|evening),/ })).toBeVisible()
}

// ── Seeded members (not the thing under test) ──
// seedMember creates an active user with one role, the way an admin would
// have set them up. A department makes the role department-scoped; a batch
// also gives them a Student identity there.
export async function seedMember(
  apiContext: APIRequestContext,
  member: { role: string; fullName: string; department?: string; batch?: number },
) {
  const account = await bootstrapAdmin('')
  await activateUserViaAPI(apiContext, account.activationToken, account.password)
  const { v4: uuidv4 } = await import('uuid')
  const client = await getSchemaClient()
  try {
    await client.query(`UPDATE users SET status = 'active' WHERE id = $1`, [account.userId])
    await client.query(`UPDATE profiles SET full_name = $2 WHERE user_id = $1`, [account.userId, member.fullName])
    const dept = member.department
      ? (await client.query('SELECT id FROM departments WHERE code = $1', [member.department])).rows[0].id
      : null
    if (member.batch && dept) {
      // USNs are unique, and other specs seed students too, so pick a roll
      // number that is still free rather than trusting a random one.
      let usn = ''
      for (let attempt = 0; attempt < 50 && !usn; attempt++) {
        const candidate = `4MN${String(member.batch).slice(2)}${member.department}${String(Math.floor(Math.random() * 900) + 100)}`
        const taken = await client.query('SELECT 1 FROM student_identities WHERE lower(usn) = lower($1)', [candidate])
        if (taken.rowCount === 0) usn = candidate
      }
      if (!usn) throw new Error('no free USN found for the seeded student')
      await client.query(
        `INSERT INTO student_identities (user_id, usn, department_id, batch_year) VALUES ($1, $2, $3, $4)`,
        [account.userId, usn, dept, member.batch],
      )
      await client.query(
        `INSERT INTO role_assignments (id, user_id, role, scope_type, starts_at, created_at) VALUES ($1, $2, $3, 'global', NOW() - interval '1 minute', NOW())`,
        [uuidv4(), account.userId, member.role],
      )
    } else if (dept) {
      await client.query(
        `INSERT INTO role_assignments (id, user_id, role, scope_type, scope_id, starts_at, created_at) VALUES ($1, $2, $3, 'department', $4, NOW() - interval '1 minute', NOW())`,
        [uuidv4(), account.userId, member.role, dept],
      )
    } else {
      await client.query(
        `INSERT INTO role_assignments (id, user_id, role, scope_type, starts_at, created_at) VALUES ($1, $2, $3, 'global', NOW() - interval '1 minute', NOW())`,
        [uuidv4(), account.userId, member.role],
      )
    }
  } finally {
    await client.end()
  }
  return { email: account.email, password: account.password, userId: account.userId }
}
