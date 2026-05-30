import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useChatStore } from '../stores/chat'

export function ChatPage() {
  const { messages, sending, error, send } = useChatStore()
  const [input, setInput] = useState('')
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const text = input.trim()
    if (!text) return
    setInput('')
    void send(text)
  }

  return (
    <div className="chat">
      <div className="messages">
        {messages.length === 0 && (
          <div className="empty muted">Ask PicoClaw anything. It can read/write files and run shell commands.</div>
        )}
        {messages.map((m) => (
          <div key={m.id} className={`msg ${m.role}`}>
            <div className="role">{m.role === 'user' ? 'You' : 'PicoClaw'}</div>
            {m.tools.map((t, i) => (
              <div key={i} className="toolcall">
                <code>{t.name}</code> <span className="muted">{t.args}</span>
                {t.result !== undefined && <div className="toolresult">{t.result}</div>}
              </div>
            ))}
            {m.text && <div className="text">{m.text}</div>}
            {m.role === 'assistant' && !m.text && sending && (
              <div className="typing">●●●</div>
            )}
          </div>
        ))}
        <div ref={endRef} />
      </div>
      {error && <div className="error bar">{error}</div>}
      <form className="composer" onSubmit={submit}>
        <input
          placeholder="Message PicoClaw…"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          disabled={sending}
        />
        <button type="submit" disabled={sending || !input.trim()}>
          Send
        </button>
      </form>
    </div>
  )
}
