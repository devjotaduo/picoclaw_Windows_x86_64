import { http } from "./http";

export interface NamedAgent {
	name: string;
	description?: string;
	system_prompt: string;
	model?: string;
	temperature?: number;
	enabled?: boolean;
}

export interface AgentsPayload {
	agents: NamedAgent[];
	models: string[];
	default: string;
}

export interface AgentInfo {
	name: string;
	description?: string;
	model?: string;
	enabled?: boolean;
}

export const agentsApi = {
	list: () => http.get<AgentsPayload>("/api/agents"),
	get: (name: string) => http.get<AgentInfo>(`/api/agents/${encodeURIComponent(name)}`),
	save: (a: NamedAgent) => http.post<{ ok: boolean; agent: NamedAgent }>("/api/agents", a),
	remove: (name: string) => http.del<{ ok: boolean }>(`/api/agents/${encodeURIComponent(name)}`),
};
