import { http } from './http'

export interface Credential {
  protocol: string
  has_key: boolean
}

export const credentialsApi = {
  list: () => http.get<{ credentials: Credential[] }>('/api/credentials'),
  set: (protocol: string, api_key: string) =>
    http.post('/api/credentials', { protocol, api_key }),
  remove: (protocol: string) =>
    http.del(`/api/credentials/${encodeURIComponent(protocol)}`),
}
