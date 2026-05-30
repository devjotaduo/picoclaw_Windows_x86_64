import { env } from './env.ts'
import type { Client } from './db.ts'

// buildConfig produces the PicoClaw config.json provisioned for a client.
// The OpenRouter API key is injected from the server environment and is the
// only secret here — it is returned for download by the admin, never shown to
// the client's browser through any client-facing surface.
export function buildConfig(client: Client): Record<string, unknown> {
  return {
    version: 1,
    workspace: client.workspace,
    model_list: [
      {
        name: env.defaultModel,
        api_key: env.openrouterKey,
      },
    ],
    agents: {
      defaults: {
        model_name: env.defaultModel,
        max_turns: 12,
      },
    },
    gateway: {
      host: '127.0.0.1',
      port: 18790,
    },
    // Identity provisioned by the SaaS admin panel.
    identity: {
      client_id: client.id,
      slug: client.slug,
      access_token: client.access_token,
    },
  }
}
