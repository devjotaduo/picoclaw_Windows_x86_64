import { http } from './http'

export interface SystemInfo {
  version: string
  workspace: string
  default_model: string
  agent_ready: boolean
}

export const systemApi = {
  info: () => http.get<SystemInfo>('/api/system'),
}
