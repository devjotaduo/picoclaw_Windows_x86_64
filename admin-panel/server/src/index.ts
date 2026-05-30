import express from 'express'
import cookieParser from 'cookie-parser'
import { env } from './env.ts'
import { authRouter } from './routes/auth.ts'
import { clientsRouter } from './routes/clients.ts'

const app = express()
app.use(express.json())
app.use(cookieParser())

// Credentialed CORS for a cross-origin frontend (e.g. Vercel), gated on
// CORS_ORIGIN so local same-origin dev is unaffected.
if (env.corsOrigin) {
  app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', env.corsOrigin)
    res.header('Access-Control-Allow-Credentials', 'true')
    res.header('Vary', 'Origin')
    if (req.method === 'OPTIONS') {
      res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
      res.header('Access-Control-Allow-Headers', 'Content-Type')
      res.header('Access-Control-Max-Age', '86400')
      res.sendStatus(204)
      return
    }
    next()
  })
}

app.get('/api/health', (_req, res) => {
  res.json({ status: 'ok' })
})

app.use('/api/auth', authRouter)
app.use('/api/clients', clientsRouter)

app.listen(env.port, () => {
  console.log(`admin-panel API on http://127.0.0.1:${env.port}`)
  if (!env.openrouterKey) {
    console.warn('warning: OPENROUTER_API_KEY is empty — provisioned configs will have no key')
  }
})
