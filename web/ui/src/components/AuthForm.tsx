import { useState, type FormEvent } from 'react'
import { launcherAuth } from '../api/launcher-auth'

// AuthForm handles both first-run setup (set a password) and login.
export function AuthForm({
  mode,
  onDone,
}: {
  mode: 'setup' | 'login'
  onDone: () => Promise<void> | void
}) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      if (mode === 'setup') await launcherAuth.setup(password)
      else await launcherAuth.login(password)
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
        <h1>PicoClaw</h1>
        <p className="muted">
          {mode === 'setup' ? 'Set a password to protect this launcher.' : 'Enter your password.'}
        </p>
        <input
          type="password"
          autoFocus
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && <p className="error">{error}</p>}
        <button type="submit" disabled={busy || password.length < 4}>
          {busy ? '…' : mode === 'setup' ? 'Create' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}
