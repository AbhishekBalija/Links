import type { Page, APIRequestContext } from '@playwright/test'
import { getSchemaClient } from './db'

// ── Admin bootstrap (seeded directly — not the thing under test) ──
export async function bootstrapAdmin(_dbURL: string) {
  void _dbURL
  const { v4: uuidv4 } = await import('uuid')
  const { randomBytes, createHash } = await import('crypto')

  const adminEmail = `admin-${Date.now()}@test.com`
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
export async function loginViaUI(page: Page, email: string, password: string) {
  await page.goto('/login')
  await page.waitForURL('**/login')
  await page.fill('#email', email)
  await page.fill('#password', password)
  await page.click('button[type="submit"]')
  await page.waitForURL((url) => !url.pathname.includes('/login'))
}

export async function submitAccessRequestViaUI(
  page: Page,
  data: { full_name: string; email: string; password: string; usn: string; department_code: string }
) {
  await page.goto('/access-request')
  await page.waitForURL('**/access-request')
  await page.fill('#full_name', data.full_name)
  await page.fill('#email', data.email)
  await page.fill('#password', data.password)
  await page.fill('#usn', data.usn)
  await page.selectOption('#department_code', data.department_code)
  await page.click('button[type="submit"]')
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

export async function getActivationToken(_dbURL: string, userId: string): Promise<string> {
  const client = await getSchemaClient()
  try {
    const result = await client.query(
      `SELECT token_hash FROM account_activation_tokens
       WHERE user_id = $1 AND purpose = 'activate' AND used_at IS NULL
       ORDER BY created_at DESC LIMIT 1`,
      [userId]
    )
    if (!result.rows[0]) throw new Error(`No activation token found for user ${userId}`)
    return result.rows[0].token_hash
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

export async function loginViaAPI(apiContext: APIRequestContext, email: string, password: string): Promise<string> {
  const res = await apiContext.post('/api/v1/auth/login', {
    data: { email, password },
  })
  if (!res.ok()) {
    const body = await res.text()
    throw new Error(`Login failed: ${body}`)
  }
  const body = await res.json()
  return body.data.access_token
}

// ── Cleanup ──
export async function cleanupTestUsers(_dbURL: string, emails: string[]) {
  const client = await getSchemaClient()
  try {
    for (const email of emails) {
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
