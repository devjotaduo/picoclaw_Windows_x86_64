// API client for the admin panel. Same-origin via the Vite dev proxy (/api) in
// dev, or a cross-origin backend when VITE_API_BASE is set (e.g. Vercel).

const API_BASE = (import.meta.env.VITE_API_BASE ?? '').replace(/\/+$/, '')
const CREDENTIALS: RequestCredentials = API_BASE ? 'include' : 'same-origin'

export interface Client {
  id: string
  name: string
  slug: string
  access_token: string
  workspace: string
  status: 'active' | 'suspended'
  created_at: string
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(API_BASE + path, {
    method,
    credentials: CREDENTIALS,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : {}
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data as T
}

export const api = {
  me: () => req<{ authed: boolean }>('GET', '/api/auth/me'),
  login: (password: string) => req('POST', '/api/auth/login', { password }),
  logout: () => req('POST', '/api/auth/logout'),

  listClients: () => req<{ clients: Client[] }>('GET', '/api/clients'),
  createClient: (name: string) => req<{ client: Client }>('POST', '/api/clients', { name }),
  deleteClient: (id: string) => req('DELETE', `/api/clients/${id}`),
  regenerateToken: (id: string) => req<{ client: Client }>('POST', `/api/clients/${id}/regenerate-token`),
  setStatus: (id: string, status: 'active' | 'suspended') =>
    req<{ client: Client }>('POST', `/api/clients/${id}/status`, { status }),
  provisionUrl: (id: string) => `${API_BASE}/api/clients/${id}/provision`,
}
