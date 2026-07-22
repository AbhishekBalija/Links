import { dropTestSchema } from './helpers/db'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const STATE_FILE = path.resolve(__dirname, '.e2e-state.json')

export default async function globalTeardown() {
  console.log('\n=== E2E Global Teardown ===')

  // Read state
  let state: { backendPid?: number; frontendPid?: number; schemaName?: string } = {}
  try {
    state = JSON.parse(fs.readFileSync(STATE_FILE, 'utf-8'))
    fs.unlinkSync(STATE_FILE)
  } catch {
    console.log('  No state file found, skipping')
  }

  // Kill backend
  if (state.backendPid) {
    try {
      process.kill(state.backendPid, 'SIGTERM')
      console.log(`  Backend (PID ${state.backendPid}) stopped`)
    } catch {
      console.log(`  Backend (PID ${state.backendPid}) already stopped`)
    }
  }

  // Kill frontend
  if (state.frontendPid) {
    try {
      process.kill(state.frontendPid, 'SIGTERM')
      console.log(`  Frontend (PID ${state.frontendPid}) stopped`)
    } catch {
      console.log(`  Frontend (PID ${state.frontendPid}) already stopped`)
    }
  }

  // Drop schema
  console.log('  Dropping test schema...')
  await dropTestSchema()

  console.log('=== Teardown complete ===\n')
}
