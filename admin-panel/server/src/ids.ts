import { randomBytes, randomUUID } from 'node:crypto'

// genId returns a short unique id.
export function genId(): string {
  return randomUUID()
}

// genToken returns a client access token, prefixed pc_ for recognisability.
export function genToken(): string {
  return 'pc_' + randomBytes(24).toString('hex')
}

// slugify turns a display name into a URL/path-safe slug.
export function slugify(name: string): string {
  const base = name
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return base || 'client'
}
