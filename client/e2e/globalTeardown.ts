import { dropTestSchema } from './helpers/db'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const STATE_FILE = path.resolve(__dirname, '.e2e-state.json')

function stopProcess(pid: number | undefined, label: string) {
  if (!pid) return
  try {
    process.kill(process.platform === 'win32' ? pid : -pid, 'SIGTERM')
    console.log(`  ${label} (PID ${pid}) stopped`)
  } catch {
    console.log(`  ${label} (PID ${pid}) already stopped`)
  }
}

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

  stopProcess(state.backendPid, 'Backend')
  stopProcess(state.frontendPid, 'Frontend')

  // Drop schema
  console.log('  Dropping test schema...')
	await dropTestSchema(state.schemaName)

  console.log('=== Teardown complete ===\n')
}
