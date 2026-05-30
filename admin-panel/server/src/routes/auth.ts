import { Router } from 'express'
import { checkPassword, setSession, clearSession, isAuthed } from '../auth.ts'

export const authRouter = Router()

authRouter.post('/login', (req, res) => {
  const { password } = req.body ?? {}
  if (!checkPassword(String(password ?? ''))) {
    res.status(401).json({ error: 'wrong password' })
    return
  }
  setSession(res)
  res.json({ ok: true })
})

authRouter.post('/logout', (_req, res) => {
  clearSession(res)
  res.json({ ok: true })
})

authRouter.get('/me', (req, res) => {
  res.json({ authed: isAuthed(req) })
})
