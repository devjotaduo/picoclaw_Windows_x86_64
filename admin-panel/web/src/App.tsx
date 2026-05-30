import { useEffect, useState } from 'react'
import { Outlet, NavLink } from 'react-router-dom'
import { api } from './api.ts'
import { LoginPage } from './pages/Login.tsx'

type Phase = 'loading' | 'login' | 'ready'

// App is the authenticated shell: it gates on the admin session, then renders
// the sidebar + routed page.
export function App() {
  const [phase, setPhase] = useState<Phase>('loading')

  const refresh = async () => {
    try {
      const { authed } = await api.me()
      setPhase(authed ? 'ready' : 'login')
    } catch {
      setPhase('login')
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  if (phase === 'loading') return <div className="center muted">Carregando…</div>
  if (phase === 'login') return <LoginPage onDone={refresh} />

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">🐾 PicoClaw <span className="tag">Admin</span></div>
        <nav>
          <NavLink to="/tenants" className="nav-link">
            Clientes
          </NavLink>
        </nav>
        <button
          className="logout"
          onClick={async () => {
            await api.logout()
            await refresh()
          }}
        >
          Sair
        </button>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  )
}
