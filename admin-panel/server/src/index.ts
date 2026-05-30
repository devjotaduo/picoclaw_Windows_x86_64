import express from 'express'
import cookieParser from 'cookie-parser'
import { env } from './env.ts'
import { authRouter } from './routes/auth.ts'
import { clientsRouter } from './routes/clients.ts'

const app = express()
app.use(express.json())
app.use(cookieParser())

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
