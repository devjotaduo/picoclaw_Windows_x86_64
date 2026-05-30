import { useEffect, useState, type ReactNode } from 'react'
import { launcherAuth } from '../api/launcher-auth'
import { Sidebar } from './Sidebar'
import { AuthForm } from './AuthForm'

type Phase = 'loading' | 'setup' | 'login' | 'ready'

export function AppShell({ children }: { children: ReactNode }) {
  const [phase, setPhase] = useState<Phase>('loading')

  const refresh = async () => {
    try {
      const s = await launcherAuth.status()
      setPhase(s.needs_setup ? 'setup' : s.authed ? 'ready' : 'login')
    } catch {
      setPhase('login')
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  if (phase === 'loading') {
    return <div className="center muted">Loading…</div>
  }

  if (phase !== 'ready') {
    return (
      <AuthForm
        mode={phase}
        onDone={async () => {
          await refresh()
        }}
      />
    )
  }

  return (
    <div className="app">
      <Sidebar
        onLogout={async () => {
          await launcherAuth.logout()
          await refresh()
        }}
      />
      <main className="content">{children}</main>
    </div>
  )
}
