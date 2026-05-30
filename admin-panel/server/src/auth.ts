import { createHmac, timingSafeEqual } from 'node:crypto'
import type { Request, Response, NextFunction } from 'express'
import { env } from './env.ts'

const COOKIE = 'admin_session'
const PAYLOAD = 'authed'

function sign(payload: string): string {
  const sig = createHmac('sha256', env.cookieSecret).update(payload).digest('hex')
  return `${payload}.${sig}`
}

function valid(token: string | undefined): boolean {
  if (!token) return false
  const expected = sign(PAYLOAD)
  if (token.length !== expected.length) return false
  return timingSafeEqual(Buffer.from(token), Buffer.from(expected))
}

export function isAuthed(req: Request): boolean {
  return valid(req.cookies?.[COOKIE])
}

export function setSession(res: Response): void {
  const crossSite = env.corsOrigin !== ''
  res.cookie(COOKIE, sign(PAYLOAD), {
    httpOnly: true,
    sameSite: crossSite ? 'none' : 'lax',
    secure: crossSite,
    maxAge: 7 * 24 * 3600 * 1000,
  })
}

export function clearSession(res: Response): void {
  const crossSite = env.corsOrigin !== ''
  res.clearCookie(COOKIE, { sameSite: crossSite ? 'none' : 'lax', secure: crossSite })
}

export function checkPassword(password: string): boolean {
  const a = Buffer.from(password || '')
  const b = Buffer.from(env.adminPassword)
  return a.length === b.length && timingSafeEqual(a, b)
}

export function requireAuth(req: Request, res: Response, next: NextFunction): void {
  if (!isAuthed(req)) {
    res.status(401).json({ error: 'not authenticated' })
    return
  }
  next()
}
