import { Router } from 'express'
import { clients } from '../db.ts'
import { buildConfig } from '../provision.ts'
import { requireAuth } from '../auth.ts'

export const clientsRouter = Router()

// All client routes require an authenticated admin.
clientsRouter.use(requireAuth)

// List all clients.
clientsRouter.get('/', (_req, res) => {
  res.json({ clients: clients.list() })
})

// Create a client (provisions id, slug, access_token, isolated workspace).
clientsRouter.post('/', (req, res) => {
  const name = String(req.body?.name ?? '').trim()
  if (!name) {
    res.status(400).json({ error: 'name is required' })
    return
  }
  res.status(201).json({ client: clients.create(name) })
})

// Delete a client.
clientsRouter.delete('/:id', (req, res) => {
  if (!clients.remove(req.params.id)) {
    res.status(404).json({ error: 'no such client' })
    return
  }
  res.json({ ok: true })
})

// Regenerate a client's access token.
clientsRouter.post('/:id/regenerate-token', (req, res) => {
  const client = clients.regenerateToken(req.params.id)
  if (!client) {
    res.status(404).json({ error: 'no such client' })
    return
  }
  res.json({ client })
})

// Toggle active/suspended.
clientsRouter.post('/:id/status', (req, res) => {
  const status = req.body?.status === 'suspended' ? 'suspended' : 'active'
  const client = clients.setStatus(req.params.id, status)
  if (!client) {
    res.status(404).json({ error: 'no such client' })
    return
  }
  res.json({ client })
})

// Download the provisioned PicoClaw config.json for a client.
clientsRouter.get('/:id/provision', (req, res) => {
  const client = clients.get(req.params.id)
  if (!client) {
    res.status(404).json({ error: 'no such client' })
    return
  }
  const config = buildConfig(client)
  res.setHeader('Content-Type', 'application/json')
  res.setHeader('Content-Disposition', `attachment; filename="picoclaw-${client.slug}.config.json"`)
  res.send(JSON.stringify(config, null, 2))
})
