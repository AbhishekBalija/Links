import { createTestSchema, sweepStaleSchemas, getDatabaseURL } from './helpers/db'
import { spawn } from 'child_process'
import * as http from 'http'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

export type SetupResult = {
  backendUrl: string
  frontendUrl: string
  schemaName: string
  backendPid: number
  frontendPid: number
}

const STATE_FILE = path.resolve(__dirname, '.e2e-state.json')

async function waitForServer(url: string, label: string, timeoutMs = 30000): Promise<void> {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    try {
      await new Promise<void>((resolve, reject) => {
        const req = http.get(url, (res) => {
          res.resume()
          resolve()
        })
        req.on('error', reject)
        req.setTimeout(2000, () => { req.destroy(); reject(new Error('timeout')) })
      })
      console.log(`  ${label} ready at ${url}`)
      return
    } catch {
      await new Promise((r) => setTimeout(r, 500))
    }
  }
  throw new Error(`${label} did not start within ${timeoutMs}ms`)
}

export default async function globalSetup(): Promise<SetupResult> {
  console.log('\n=== E2E Global Setup ===')

  // 1. Create isolated schema and sweep stale ones
  console.log('[1] Creating test schema...')
  const schemaName = await createTestSchema()
  process.env.E2E_SCHEMA_NAME = schemaName
  console.log(`  Schema: ${schemaName}`)
  console.log('[2] Sweeping stale schemas (>24h)...')
  await sweepStaleSchemas(schemaName)

  const dbURL = getDatabaseURL()

  // 2. Start backend server
  // CI pre-builds the binary — globalSetup still creates the schema first,
  // then starts the binary with the correct per-schema DATABASE_URL.
  const backendBin = process.env.E2E_BACKEND_BIN || 'go'
  const backendArgs = backendBin === 'go' ? ['run', './cmd/api'] : []
  console.log(`[3] Starting backend (${backendBin} ${backendArgs.join(' ')})...`)
  const backendProcess = spawn(backendBin, backendArgs, {
    cwd: path.resolve(__dirname, '../../server'),
    env: {
      ...process.env,
      DATABASE_URL: dbURL,
      APP_PORT: '8081',
      RESEND_API_KEY: '',
      ACCESS_TOKEN_TTL: '10s',
      CORS_ALLOWED_ORIGINS: 'http://localhost:5174',
    },
    stdio: 'pipe',
  })

  backendProcess.stdout?.on('data', (d) => process.stdout.write(`[b] ${d}`))
  backendProcess.stderr?.on('data', (d) => process.stderr.write(`[b] ${d}`))

  await waitForServer('http://localhost:8081/api/health', 'Backend')

  // 3. Start frontend dev server
  console.log('[4] Starting frontend...')
  const frontendProcess = spawn('bun', ['run', 'dev', '--port', '5174'], {
    cwd: path.resolve(__dirname, '..'),
    env: {
      ...process.env,
      VITE_API_URL: 'http://localhost:8081',
    },
    stdio: 'pipe',
  })

  frontendProcess.stdout?.on('data', (d) => process.stdout.write(`[f] ${d}`))
  frontendProcess.stderr?.on('data', (d) => process.stderr.write(`[f] ${d}`))

  await waitForServer('http://localhost:5174', 'Frontend')

  // 4. Save state for teardown
  const state: SetupResult = {
    backendUrl: 'http://localhost:8081',
    frontendUrl: 'http://localhost:5174',
    schemaName,
    backendPid: backendProcess?.pid ?? 0,
    frontendPid: frontendProcess.pid!,
  }
  fs.writeFileSync(STATE_FILE, JSON.stringify(state, null, 2))

  console.log('=== Setup complete ===\n')
  return state
}
