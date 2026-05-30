import { Link } from '@tanstack/react-router'

interface NavItem {
  to: string
  label: string
  group?: string
}

const NAV: NavItem[] = [
  { to: '/', label: 'Chat' },
  { to: '/agents', label: 'Agentes' },
  { to: '/whatsapp', label: 'WhatsApp' },
  { to: '/models', label: 'Models', group: 'Models' },
  { to: '/credentials', label: 'Credentials', group: 'Models' },
]

export function Sidebar({ onLogout }: { onLogout: () => void }) {
  return (
    <aside className="sidebar">
      <div className="brand">🐾 PicoClaw</div>
      <nav>
        {NAV.map((item) => (
          <Link
            key={item.to}
            to={item.to}
            className="nav-link"
            activeProps={{ className: 'nav-link active' }}
            activeOptions={{ exact: item.to === '/' }}
          >
            {item.label}
          </Link>
        ))}
      </nav>
      <button className="logout" onClick={onLogout}>
        Sign out
      </button>
    </aside>
  )
}
