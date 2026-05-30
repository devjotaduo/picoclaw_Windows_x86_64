import { create } from 'zustand'
import { streamChat, type ChatEvent } from '../api/chat'

export interface ToolCall {
  name: string
  args: string
  result?: string
}

export interface Message {
  id: number
  role: 'user' | 'assistant'
  text: string
  tools: ToolCall[]
}

interface ChatState {
  messages: Message[]
  sending: boolean
  error: string | null
  send: (text: string) => Promise<void>
  clear: () => void
}

let nextId = 1

export const useChatStore = create<ChatState>((set, get) => ({
  messages: [],
  sending: false,
  error: null,

  clear: () => set({ messages: [], error: null }),

  send: async (text: string) => {
    if (get().sending || !text.trim()) return

    const userMsg: Message = { id: nextId++, role: 'user', text, tools: [] }
    const assistantMsg: Message = { id: nextId++, role: 'assistant', text: '', tools: [] }
    set((s) => ({
      messages: [...s.messages, userMsg, assistantMsg],
      sending: true,
      error: null,
    }))

    const patch = (fn: (m: Message) => Message) =>
      set((s) => ({
        messages: s.messages.map((m) => (m.id === assistantMsg.id ? fn(m) : m)),
      }))

    const onEvent = (e: ChatEvent) => {
      switch (e.type) {
        case 'tool_call':
          patch((m) => ({ ...m, tools: [...m.tools, { name: e.name!, args: e.args || '' }] }))
          break
        case 'tool_result':
          patch((m) => {
            const tools = [...m.tools]
            for (let i = tools.length - 1; i >= 0; i--) {
              if (tools[i].name === e.name && tools[i].result === undefined) {
                tools[i] = { ...tools[i], result: e.result }
                break
              }
            }
            return { ...m, tools }
          })
          break
        case 'assistant':
          patch((m) => ({ ...m, text: e.text || '' }))
          break
        case 'error':
          set({ error: e.text || 'error' })
          patch((m) => ({ ...m, text: m.text || `(error: ${e.text})` }))
          break
      }
    }

    try {
      await streamChat(text, onEvent)
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) })
    } finally {
      set({ sending: false })
    }
  },
}))
