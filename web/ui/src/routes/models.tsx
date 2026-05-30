import { useEffect, useState, type FormEvent } from 'react'
import { modelsApi, type Model } from '../api/models'

export function ModelsPage() {
  const [models, setModels] = useState<Model[]>([])
  const [defaultModel, setDefaultModel] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [apiKey, setApiKey] = useState('')

  const load = async () => {
    try {
      const res = await modelsApi.list()
      setModels(res.models)
      setDefaultModel(res.default_model)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const add = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    try {
      await modelsApi.add({ name, base_url: baseURL || undefined, api_key: apiKey || undefined })
      setName('')
      setBaseURL('')
      setApiKey('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <div className="page">
      <h1>Models</h1>
      {error && <p className="error">{error}</p>}

      <div className="cards">
        {models.map((m) => (
          <div key={m.name} className={`card model ${m.name === defaultModel ? 'is-default' : ''}`}>
            <div className="model-name">{m.name}</div>
            <div className="muted small">{m.base_url || 'default endpoint'}</div>
            <div className="row">
              <span className={`badge ${m.has_key ? 'ok' : 'warn'}`}>
                {m.has_key ? 'key set' : 'no key'}
              </span>
              {m.name === defaultModel ? (
                <span className="badge ok">default</span>
              ) : (
                <button onClick={() => modelsApi.setDefault(m.name).then(load)}>Set default</button>
              )}
              <button className="danger" onClick={() => modelsApi.remove(m.name).then(load)}>
                Delete
              </button>
            </div>
          </div>
        ))}
        {models.length === 0 && <p className="muted">No models yet. Add one below.</p>}
      </div>

      <form className="card form" onSubmit={add}>
        <h2>Add model</h2>
        <input placeholder="protocol/model (e.g. openai/gpt-4o-mini)" value={name} onChange={(e) => setName(e.target.value)} />
        <input placeholder="base_url (optional)" value={baseURL} onChange={(e) => setBaseURL(e.target.value)} />
        <input placeholder="api_key (optional)" type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
        <button type="submit" disabled={!name.includes('/')}>
          Add
        </button>
      </form>
    </div>
  )
}
