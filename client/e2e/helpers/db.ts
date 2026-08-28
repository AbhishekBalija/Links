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

let schemaName = process.env.E2E_SCHEMA_NAME || ''

export function getSchemaName() {
	return schemaName
}

// Returns the unpooled URL with search_path via options parameter.
// Used by globalSetup.ts as DATABASE_URL env var — the backend's GORM
// handles the options startup parameter correctly.
export function getDatabaseURL() {
	if (!schemaName) throw new Error('E2E schema has not been initialized')
	const url = new URL(getNeonURLs().unpooled)
	url.searchParams.set('options', `--search_path=${schemaName}`)
	return url.toString()
}

export function getBaseURL() {
	return getNeonURLs().pooled
}

// Creates a pg.Client connected to the test schema using the unpooled
// endpoint + explicit SET search_path (pg driver doesn't reliably forward
// the options startup parameter).
export async function getSchemaClient() {
	const client = new Client({ connectionString: getNeonURLs().unpooled })
  await client.connect()
  await client.query(`SET search_path TO "${schemaName}"`)
  return client
}

export async function getPooledClient() {
	const client = new Client({ connectionString: getNeonURLs().pooled })
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
	const ts = new Date().toISOString().replaceAll('-', '').replaceAll(':', '').replace('T', '_').slice(0, 15)
	schemaName = `e2e_test_${ts}_${randomBytes(3).toString('hex')}`

  const client = await getPooledClient()
  try {
    await client.query(`CREATE SCHEMA IF NOT EXISTS "${schemaName}"`)
  } finally {
    await client.end()
  }

  return schemaName
}

export async function dropTestSchema(name = schemaName) {
	if (!name) return
	const client = await getPooledClient()
	try {
		await client.query(`DROP SCHEMA IF EXISTS "${name}" CASCADE`)
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
		await client.query(`DELETE FROM audit_logs WHERE actor_id IN (SELECT id FROM users WHERE email = $1) OR resource_id IN (SELECT id FROM users WHERE email = $1)`, [email])
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
