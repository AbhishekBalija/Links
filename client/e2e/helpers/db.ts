import { Client } from 'pg'
import { v4 as uuidv4 } from 'uuid'
import { randomBytes, createHash } from 'crypto'

function getNeonURLs(): { pooled: string; unpooled: string } {
  const pooled = process.env.E2E_NEON_URL || process.env.DATABASE_URL
  if (!pooled) {
    throw new Error(
      'E2E_NEON_URL (or fallback DATABASE_URL) must be set. ' +
      'Use the pooled Neon connection string (contains -pooler). ' +
      'Example: E2E_NEON_URL=postgresql://user:pass@host-pooler.region.neon.tech/db?sslmode=require'
    )
  }
  return {
    pooled,
    unpooled: pooled.replace('-pooler', ''),
  }
}

const URLs = getNeonURLs()

// Pooled connection — for schema-level operations (CREATE/DROP SCHEMA) which
// don't need search_path. The pooler rejects the options startup parameter.
const POOLED_NEON_URL = URLs.pooled

// Unpooled (direct) connection — supports the options/search_path startup
// parameter needed for per-schema migrations.
const UNPOOLED_NEON_URL = URLs.unpooled

let schemaName = process.env.E2E_SCHEMA_NAME || ''

export function getSchemaName() {
  return schemaName
}

// Returns the unpooled URL with search_path via options parameter.
// Used by globalSetup.ts as DATABASE_URL env var — the backend's GORM
// handles the options startup parameter correctly.
export function getDatabaseURL() {
  const encoded = `--search_path%3D${schemaName}`
  return `${UNPOOLED_NEON_URL}&options=${encoded}`
}

export function getBaseURL() {
  return POOLED_NEON_URL
}

// Creates a pg.Client connected to the test schema using the unpooled
// endpoint + explicit SET search_path (pg driver doesn't reliably forward
// the options startup parameter).
export async function getSchemaClient() {
  const client = new Client({ connectionString: UNPOOLED_NEON_URL })
  await client.connect()
  await client.query(`SET search_path TO "${schemaName}"`)
  return client
}

export async function getPooledClient() {
  const client = new Client({ connectionString: POOLED_NEON_URL })
  await client.connect()
  return client
}

export async function sweepStaleSchemas(exceptName?: string) {
  const client = await getPooledClient()
  try {
    const result = await client.query(
      `SELECT schema_name FROM information_schema.schemata
       WHERE schema_name LIKE 'e2e_test_%'`
    )
    const now = Date.now()
    for (const row of result.rows) {
      const name = row.schema_name as string
      if (name === exceptName) continue
      // Schema names: e2e_test_YYYYMMDD_HHMMSS
      const tsStr = name.replace('e2e_test_', '')
      const year = tsStr.slice(0, 4)
      const month = tsStr.slice(4, 6)
      const day = tsStr.slice(6, 8)
      const hour = tsStr.slice(9, 11)
      const min = tsStr.slice(11, 13)
      const sec = tsStr.slice(13, 15)
      const schemaDate = Date.parse(`${year}-${month}-${day}T${hour}:${min}:${sec}Z`)
      if (!isNaN(schemaDate) && schemaDate < now - 86400000) {
        console.log(`  Dropping stale schema: ${name}`)
        await client.query(`DROP SCHEMA IF EXISTS "${name}" CASCADE`)
      }
    }
  } finally {
    await client.end()
  }
}

export async function createTestSchema(): Promise<string> {
  const now = new Date()
  const ts = `${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}_${String(now.getHours()).padStart(2, '0')}${String(now.getMinutes()).padStart(2, '0')}${String(now.getSeconds()).padStart(2, '0')}`
  schemaName = `e2e_test_${ts}`

  const client = await getPooledClient()
  try {
    await client.query(`CREATE SCHEMA IF NOT EXISTS "${schemaName}"`)
  } finally {
    await client.end()
  }

  return schemaName
}

export async function dropTestSchema() {
  if (!schemaName) return
  const client = await getPooledClient()
  try {
    await client.query(`DROP SCHEMA IF EXISTS "${schemaName}" CASCADE`)
  } finally {
    await client.end()
  }
}

export type TestUserData = {
  email: string
  password: string
  userId: string
  activationToken: string
}

export async function seedTestUser(email: string, passwordPlain: string): Promise<TestUserData> {
  const client = await getSchemaClient()
  try {
    const userId = uuidv4()
    const profileId = userId
    const now = new Date().toISOString()

    // Create user (status: pending)
    await client.query(
      `INSERT INTO users (id, email, password_hash, status, is_verified, created_at, updated_at)
       VALUES ($1, $2, $3, 'pending', false, $4, $4)`,
      [userId, email, passwordPlain, now]
    )

    // Create profile
    const username = `test_${uuidv4().slice(0, 8)}`
    await client.query(
      `INSERT INTO profiles (user_id, username, full_name, created_at, updated_at)
       VALUES ($1, $2, $3, $4, $4)`,
      [profileId, username, 'E2E Test User', now]
    )

    // Create activation token
    const tokenRaw = randomBytes(32).toString('base64url')
    const tokenHash = createHash('sha256').update(tokenRaw).digest('hex')
    await client.query(
      `INSERT INTO account_activation_tokens (id, user_id, token_hash, purpose, expires_at, created_at)
       VALUES ($1, $2, $3, 'activate', $4, $5)`,
      [uuidv4(), userId, tokenHash, new Date(Date.now() + 7 * 86400000).toISOString(), now]
    )

    return { email, password: passwordPlain, userId, activationToken: tokenRaw }
  } finally {
    await client.end()
  }
}

// replaceActivationToken creates a known raw token for the activation endpoint
// after the approval path has created its production token.
export async function replaceActivationToken(userId: string): Promise<string> {
  const client = await getSchemaClient()
  try {
    const existingToken = await client.query<{ exists: boolean }>(
      'SELECT EXISTS(SELECT 1 FROM account_activation_tokens WHERE user_id = $1) AS exists',
      [userId],
    )
    if (!existingToken.rows[0]?.exists) {
      throw new Error('approval did not create an activation token')
    }

    const tokenRaw = randomBytes(32).toString('base64url')
    const tokenHash = createHash('sha256').update(tokenRaw).digest('hex')
    const now = new Date().toISOString()

    await client.query('DELETE FROM account_activation_tokens WHERE user_id = $1', [userId])
    await client.query(
      `INSERT INTO account_activation_tokens (id, user_id, token_hash, purpose, expires_at, created_at)
       VALUES ($1, $2, $3, 'activate', $4, $5)`,
      [uuidv4(), userId, tokenHash, new Date(Date.now() + 7 * 86400000).toISOString(), now],
    )

    return tokenRaw
  } finally {
    await client.end()
  }
}

export async function activateUser(userId: string) {
  const client = await getSchemaClient()
  try {
    await client.query(`UPDATE users SET status = 'active' WHERE id = $1`, [userId])
  } finally {
    await client.end()
  }
}

export async function assignRole(userId: string, role: string) {
  const client = await getSchemaClient()
  try {
    await client.query(
      `INSERT INTO role_assignments (id, user_id, role, scope_type, starts_at, created_at)
       VALUES ($1, $2, $3, 'global', NOW(), NOW())`,
      [uuidv4(), userId, role]
    )
  } finally {
    await client.end()
  }
}

export async function cleanupTestUser(email: string) {
  const client = await getSchemaClient()
  try {
    await client.query(`DELETE FROM role_assignments WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
    await client.query(`DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
    await client.query(`DELETE FROM account_activation_tokens WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
    await client.query(`DELETE FROM student_identities WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
    await client.query(`DELETE FROM profiles WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, [email])
    await client.query(`DELETE FROM users WHERE email = $1`, [email])
  } finally {
    await client.end()
  }
}
