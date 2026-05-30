// Minimal .env loader (no dependency): reads KEY=VALUE lines from ./.env if
// present, without overriding variables already set in the environment.
import { readFileSync, existsSync } from 'node:fs'
import { resolve } from 'node:path'

function loadDotenv(): void {
  const path = resolve(process.cwd(), '.env')
  if (!existsSync(path)) return
  for (const line of readFileSync(path, 'utf8').split('\n')) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue
    const eq = trimmed.indexOf('=')
    if (eq === -1) continue
    const key = trimmed.slice(0, eq).trim()
    const value = trimmed.slice(eq + 1).trim()
    if (!(key in process.env)) process.env[key] = value
  }
}

loadDotenv()

export const env = {
  port: Number(process.env.PORT) || 4000,
  adminPassword: process.env.ADMIN_PASSWORD || 'admin',
  cookieSecret: process.env.COOKIE_SECRET || 'change-me-please',
  openrouterKey: process.env.OPENROUTER_API_KEY || '',
  defaultModel: process.env.DEFAULT_MODEL || 'openrouter/openai/gpt-4o-mini',
  workspacesRoot: process.env.WORKSPACES_ROOT || './data/workspaces',
  dataDir: process.env.DATA_DIR || './data',
  // When CORS_ORIGIN is set (e.g. the Vercel frontend URL), enable credentialed
  // CORS for that origin and switch the session cookie to SameSite=None;Secure.
  corsOrigin: process.env.CORS_ORIGIN || '',
}
