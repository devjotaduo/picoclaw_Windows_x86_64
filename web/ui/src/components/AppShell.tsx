import { useEffect, useState, type ReactNode } from 'react'
import { useRouterState } from '@tanstack/react-router'
import { launcherAuth } from '../api/launcher-auth'
import { Sidebar } from './Sidebar'
import { AuthForm } from './AuthForm'

type Phase = 'loading' | 'setup' | 'login' | 'ready'

export function AppShell({ children }: { children: ReactNode }) {
  const [phase, setPhase] = useState<Phase>('loading')
  // Isolated agent pages (/a/<name>) drop the admin sidebar — they are scoped
  // to a single agent, still behind the launcher login.
  const isolated = useRouterState({ select: (s) => s.location.pathname.startsWith('/a/') })

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

  if (isolated) {
    return <main className="content isolated-agent">{children}</main>
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
