import { useState, type FormEvent } from 'react'
import { api } from '../api.ts'

export function LoginPage({ onDone }: { onDone: () => void | Promise<void> }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.login(password)
      await onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="center">
      <form className="card auth-card" onSubmit={submit}>
        <h1>PicoClaw Admin</h1>
        <p className="muted">Painel de gestão de clientes.</p>
        <input
          type="password"
          autoFocus
          placeholder="Senha de admin"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && <p className="error">{error}</p>}
        <button type="submit" disabled={busy || !password}>
          {busy ? '…' : 'Entrar'}
        </button>
      </form>
    </div>
  )
}
