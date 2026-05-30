import { http } from './http'

export interface AuthStatus {
  needs_setup: boolean
  authed: boolean
}

export const launcherAuth = {
  status: () => http.get<AuthStatus>('/api/launcher/status'),
  setup: (password: string) => http.post('/api/launcher/setup', { password }),
  login: (password: string) => http.post('/api/launcher/login', { password }),
  logout: () => http.post('/api/launcher/logout'),
}
