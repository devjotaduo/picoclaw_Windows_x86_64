import { http } from "./http";

export interface WhatsAppStatus {
	available: boolean;
	state?: "disconnected" | "qr" | "connected";
	qr?: string;
	registered?: boolean;
}

export const whatsappApi = {
	status: () => http.get<WhatsAppStatus>("/api/whatsapp/status"),
	connect: () => http.post<{ ok: boolean }>("/api/whatsapp/connect"),
	logout: () => http.post<{ ok: boolean }>("/api/whatsapp/logout"),
};
