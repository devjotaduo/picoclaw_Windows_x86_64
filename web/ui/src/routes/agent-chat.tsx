// Isolated agent page: a chat that talks to a single named agent. The backend
// builds that agent on the fly (its own name, prompt and model) so it presents
// and attends as the name it was given. Rendered without the admin sidebar.

import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useParams } from '@tanstack/react-router'
import { streamChat, type ChatEvent } from '../api/chat'
import { agentsApi, type AgentInfo } from '../api/agents'

interface ToolCall {
  name: string
  args: string
  result?: string
}

interface Message {
  id: number
  role: 'user' | 'assistant'
  text: string
  tools: ToolCall[]
}

let nextId = 1

export function AgentChatPage() {
  const { name } = useParams({ strict: false }) as { name: string }
  const [info, setInfo] = useState<AgentInfo | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let alive = true
    agentsApi
      .get(name)
      .then((a) => alive && setInfo(a))
      .catch((e) => alive && setLoadError(e instanceof Error ? e.message : String(e)))
    return () => {
      alive = false
    }
  }, [name])

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const title = info?.name || name

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const text = input.trim()
    if (!text || sending) return
    setInput('')

    const userMsg: Message = { id: nextId++, role: 'user', text, tools: [] }
    const assistantMsg: Message = { id: nextId++, role: 'assistant', text: '', tools: [] }
    setMessages((m) => [...m, userMsg, assistantMsg])
    setSending(true)
    setError(null)

    const patch = (fn: (m: Message) => Message) =>
      setMessages((ms) => ms.map((m) => (m.id === assistantMsg.id ? fn(m) : m)))

    const onEvent = (ev: ChatEvent) => {
      switch (ev.type) {
        case 'tool_call':
          patch((m) => ({ ...m, tools: [...m.tools, { name: ev.name!, args: ev.args || '' }] }))
          break
        case 'tool_result':
          patch((m) => {
            const tools = [...m.tools]
            for (let i = tools.length - 1; i >= 0; i--) {
              if (tools[i].name === ev.name && tools[i].result === undefined) {
                tools[i] = { ...tools[i], result: ev.result }
                break
              }
            }
            return { ...m, tools }
          })
          break
        case 'assistant':
          patch((m) => ({ ...m, text: ev.text || '' }))
          break
        case 'error':
          setError(ev.text || 'error')
          patch((m) => ({ ...m, text: m.text || `(error: ${ev.text})` }))
          break
      }
    }

    try {
      await streamChat(text, onEvent, undefined, name)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="chat agent-chat">
      <header className="agent-chat-head">
        <div className="agent-avatar">🐾</div>
        <div>
          <div className="agent-chat-name">{title}</div>
          {info?.description && <div className="muted agent-chat-desc">{info.description}</div>}
        </div>
      </header>

      {loadError && <div className="error bar">Agente indisponível: {loadError}</div>}

      <div className="messages">
        {messages.length === 0 && !loadError && (
          <div className="empty muted">Converse com {title}. Este chat fala só com este agente.</div>
        )}
        {messages.map((m) => (
          <div key={m.id} className={`msg ${m.role}`}>
            <div className="role">{m.role === 'user' ? 'Você' : title}</div>
            {m.tools.map((t, i) => (
              <div key={i} className="toolcall">
                <code>{t.name}</code> <span className="muted">{t.args}</span>
                {t.result !== undefined && <div className="toolresult">{t.result}</div>}
              </div>
            ))}
            {m.text && <div className="text">{m.text}</div>}
            {m.role === 'assistant' && !m.text && sending && <div className="typing">●●●</div>}
          </div>
        ))}
        <div ref={endRef} />
      </div>

      {error && <div className="error bar">{error}</div>}

      <form className="composer" onSubmit={submit}>
        <input
          placeholder={`Mensagem para ${title}…`}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          disabled={sending || !!loadError}
        />
        <button type="submit" disabled={sending || !input.trim() || !!loadError}>
          Enviar
        </button>
      </form>
    </div>
  )
}
