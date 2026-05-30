// Resolves the API origin. Empty string keeps requests same-origin (local dev
// via the Vite proxy, or the Go launcher serving the embedded UI). Set
// VITE_API_BASE at build time to point a cross-origin deploy (Vercel) at the
// backend; cross-origin requests then send credentials with `include`.

export const API_BASE = (import.meta.env.VITE_API_BASE ?? '').replace(/\/+$/, '')
export const API_CREDENTIALS: RequestCredentials = API_BASE ? 'include' : 'same-origin'

/** Prefixes a relative API path with the configured base. */
export function apiUrl(path: string): string {
  return API_BASE + path
}
