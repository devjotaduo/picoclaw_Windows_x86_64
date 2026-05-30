import { http } from './http'

export interface Model {
  name: string
  base_url?: string
  has_key: boolean
}

export interface ModelsResponse {
  models: Model[]
  default_model: string
}

export interface AddModelInput {
  name: string
  base_url?: string
  api_key?: string
  set_default?: boolean
}

export const modelsApi = {
  list: () => http.get<ModelsResponse>('/api/models'),
  add: (input: AddModelInput) => http.post('/api/models', input),
  remove: (name: string) => http.del(`/api/models/${encodeURIComponent(name)}`),
  setDefault: (name: string) => http.put(`/api/models/${encodeURIComponent(name)}`),
}
