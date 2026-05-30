import { useEffect, useState, type FormEvent } from 'react'
import { credentialsApi, type Credential } from '../api/credentials'

export function CredentialsPage() {
  const [creds, setCreds] = useState<Credential[]>([])
  const [error, setError] = useState<string | null>(null)
  const [protocol, setProtocol] = useState('')
  const [apiKey, setApiKey] = useState('')

  const load = async () => {
    try {
      setCreds((await credentialsApi.list()).credentials)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const save = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    try {
      await credentialsApi.set(protocol, apiKey)
      setProtocol('')
      setApiKey('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <div className="page">
      <h1>Credentials</h1>
      <p className="muted">API keys are stored only in the launcher's config and never sent to the browser.</p>
      {error && <p className="error">{error}</p>}

      <div className="cards">
        {creds.map((c) => (
          <div key={c.protocol} className="card cred">
            <div className="model-name">{c.protocol}</div>
            <div className="row">
              <span className={`badge ${c.has_key ? 'ok' : 'warn'}`}>
                {c.has_key ? 'key set' : 'no key'}
              </span>
              {c.has_key && (
                <button className="danger" onClick={() => credentialsApi.remove(c.protocol).then(load)}>
                  Remove
                </button>
              )}
            </div>
          </div>
        ))}
        {creds.length === 0 && <p className="muted">No providers yet.</p>}
      </div>

      <form className="card form" onSubmit={save}>
        <h2>Set credential</h2>
        <input placeholder="protocol (e.g. openai)" value={protocol} onChange={(e) => setProtocol(e.target.value)} />
        <input placeholder="api_key" type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
        <button type="submit" disabled={!protocol || !apiKey}>
          Save
        </button>
      </form>
    </div>
  )
}
