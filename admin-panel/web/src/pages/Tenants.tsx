import { useEffect, useState, type FormEvent } from 'react'
import { api, type Client } from '../api.ts'

export function TenantsPage() {
  const [clients, setClients] = useState<Client[]>([])
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const load = async () => {
    try {
      setClients((await api.listClients()).clients)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const create = async (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    setBusy(true)
    setError(null)
    try {
      await api.createClient(name.trim())
      setName('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <div>
          <h1>Clientes</h1>
          <p className="muted">Cada cliente tem um PicoClaw isolado, com workspace e token próprios.</p>
        </div>
      </header>

      <form className="card create-form" onSubmit={create}>
        <input
          placeholder="Nome do cliente (ex.: Acme Corp)"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button type="submit" disabled={busy || !name.trim()}>
          Cadastrar cliente
        </button>
      </form>

      {error && <p className="error">{error}</p>}

      <div className="clients">
        {clients.map((c) => (
          <ClientCard key={c.id} client={c} onChange={load} />
        ))}
        {clients.length === 0 && <p className="muted">Nenhum cliente cadastrado ainda.</p>}
      </div>
    </div>
  )
}

function ClientCard({ client, onChange }: { client: Client; onChange: () => void }) {
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState(false)

  const copyToken = async () => {
    await navigator.clipboard.writeText(client.access_token)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const masked = client.access_token.slice(0, 6) + '••••••••••••' + client.access_token.slice(-4)

  return (
    <div className={`card client ${client.status === 'suspended' ? 'suspended' : ''}`}>
      <div className="client-head">
        <div>
          <span className="client-name">{client.name}</span>
          <span className="muted small"> · {client.slug}</span>
        </div>
        <span className={`badge ${client.status === 'active' ? 'ok' : 'warn'}`}>{client.status}</span>
      </div>

      <div className="field">
        <label>Access token</label>
        <div className="token-row">
          <code className="token">{revealed ? client.access_token : masked}</code>
          <button className="ghost" onClick={() => setRevealed((v) => !v)}>
            {revealed ? 'ocultar' : 'ver'}
          </button>
          <button className="ghost" onClick={copyToken}>
            {copied ? 'copiado!' : 'copiar'}
          </button>
        </div>
      </div>

      <div className="field">
        <label>Workspace isolado</label>
        <code className="muted small">{client.workspace}</code>
      </div>

      <div className="actions">
        <a className="btn" href={api.provisionUrl(client.id)}>
          Baixar config
        </a>
        <button
          className="ghost"
          onClick={async () => {
            await api.regenerateToken(client.id)
            onChange()
          }}
        >
          Regenerar token
        </button>
        <button
          className="ghost"
          onClick={async () => {
            await api.setStatus(client.id, client.status === 'active' ? 'suspended' : 'active')
            onChange()
          }}
        >
          {client.status === 'active' ? 'Suspender' : 'Reativar'}
        </button>
        <button
          className="ghost danger"
          onClick={async () => {
            if (confirm(`Remover ${client.name}? Esta ação não pode ser desfeita.`)) {
              await api.deleteClient(client.id)
              onChange()
            }
          }}
        >
          Remover
        </button>
      </div>
    </div>
  )
}
