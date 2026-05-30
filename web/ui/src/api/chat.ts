// Chat transport over Server-Sent Events. The launcher streams the agent loop
// (tool calls, results, final assistant text) as `data:` lines; EventSource is
// GET-only, so we read the POST response body as a stream ourselves.

import { apiUrl, API_CREDENTIALS } from './base'

export interface ChatEvent {
  type: 'tool_call' | 'tool_result' | 'assistant' | 'done' | 'error'
  name?: string
  args?: string
  text?: string
  result?: string
}

export async function streamChat(
  message: string,
  onEvent: (e: ChatEvent) => void,
  signal?: AbortSignal,
  agent?: string,
): Promise<void> {
  const res = await fetch(apiUrl('/api/chat/stream'), {
    method: 'POST',
    credentials: API_CREDENTIALS,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(agent ? { message, agent } : { message }),
    signal,
  })
  if (!res.ok || !res.body) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `chat failed (${res.status})`)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    // SSE frames are separated by a blank line.
    let sep: number
    while ((sep = buffer.indexOf('\n\n')) !== -1) {
      const frame = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      const line = frame.split('\n').find((l) => l.startsWith('data:'))
      if (!line) continue
      try {
        onEvent(JSON.parse(line.slice(5).trim()) as ChatEvent)
      } catch {
        // ignore malformed frame
      }
    }
  }
}
