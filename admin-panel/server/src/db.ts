// Data layer backed by Node's built-in SQLite (node:sqlite, Node 22+/24).
// Run the process with --experimental-sqlite (see package.json scripts).
import { DatabaseSync } from 'node:sqlite'
import { mkdirSync } from 'node:fs'
import { resolve, join } from 'node:path'
import { env } from './env.ts'
import { genId, genToken, slugify } from './ids.ts'

export interface Client {
  id: string
  name: string
  slug: string
  access_token: string
  workspace: string
  status: 'active' | 'suspended'
  created_at: string
}

mkdirSync(resolve(env.dataDir), { recursive: true })
const db = new DatabaseSync(resolve(env.dataDir, 'admin.db'))

db.exec(`
  CREATE TABLE IF NOT EXISTS clients (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    workspace    TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   TEXT NOT NULL
  );
`)

function uniqueSlug(name: string): string {
  const base = slugify(name)
  let slug = base
  let n = 1
  const exists = db.prepare('SELECT 1 FROM clients WHERE slug = ?')
  while (exists.get(slug)) {
    n += 1
    slug = `${base}-${n}`
  }
  return slug
}

export const clients = {
  list(): Client[] {
    return db.prepare('SELECT * FROM clients ORDER BY created_at DESC').all() as unknown as Client[]
  },

  get(id: string): Client | undefined {
    return db.prepare('SELECT * FROM clients WHERE id = ?').get(id) as unknown as Client | undefined
  },

  create(name: string): Client {
    const id = genId()
    const slug = uniqueSlug(name)
    const client: Client = {
      id,
      name: name.trim(),
      slug,
      access_token: genToken(),
      workspace: join(env.workspacesRoot, slug).replace(/\\/g, '/'),
      status: 'active',
      created_at: new Date().toISOString(),
    }
    db.prepare(
      `INSERT INTO clients (id, name, slug, access_token, workspace, status, created_at)
       VALUES (?, ?, ?, ?, ?, ?, ?)`,
    ).run(
      client.id,
      client.name,
      client.slug,
      client.access_token,
      client.workspace,
      client.status,
      client.created_at,
    )
    // Provision the isolated workspace directory.
    mkdirSync(resolve(client.workspace), { recursive: true })
    return client
  },

  remove(id: string): boolean {
    const res = db.prepare('DELETE FROM clients WHERE id = ?').run(id)
    return res.changes > 0
  },

  regenerateToken(id: string): Client | undefined {
    const token = genToken()
    const res = db.prepare('UPDATE clients SET access_token = ? WHERE id = ?').run(token, id)
    if (res.changes === 0) return undefined
    return this.get(id)
  },

  setStatus(id: string, status: 'active' | 'suspended'): Client | undefined {
    const res = db.prepare('UPDATE clients SET status = ? WHERE id = ?').run(status, id)
    if (res.changes === 0) return undefined
    return this.get(id)
  },
}
