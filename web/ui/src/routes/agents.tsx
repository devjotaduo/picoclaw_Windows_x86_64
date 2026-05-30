import { useEffect, useState } from "react";
import { agentsApi, type NamedAgent } from "../api/agents";

const EMPTY: NamedAgent = {
	name: "",
	description: "",
	system_prompt: "",
	model: "",
	temperature: 0.7,
};

export function AgentsPage() {
	const [agents, setAgents] = useState<NamedAgent[]>([]);
	const [models, setModels] = useState<string[]>([]);
	const [form, setForm] = useState<NamedAgent>(EMPTY);
	const [editing, setEditing] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [msg, setMsg] = useState<string | null>(null);
	const [busy, setBusy] = useState(false);

	async function load() {
		try {
			const data = await agentsApi.list();
			setAgents(data.agents || []);
			setModels(data.models || []);
		} catch (e) {
			setError(e instanceof Error ? e.message : String(e));
		}
	}

	useEffect(() => {
		load();
	}, []);

	function newAgent() {
		setForm(EMPTY);
		setEditing(false);
		setMsg(null);
		setError(null);
	}

	function edit(a: NamedAgent) {
		setForm({ ...EMPTY, ...a });
		setEditing(true);
		setMsg(null);
		setError(null);
	}

	function set<K extends keyof NamedAgent>(key: K, value: NamedAgent[K]) {
		setForm((f) => ({ ...f, [key]: value }));
	}

	async function save(e: React.FormEvent) {
		e.preventDefault();
		if (!form.name.trim()) {
			setError("Nome é obrigatório.");
			return;
		}
		setBusy(true);
		setError(null);
		try {
			await agentsApi.save(form);
			setMsg("Salvo ✓");
			setEditing(true);
			await load();
		} catch (e) {
			setError(e instanceof Error ? e.message : String(e));
		} finally {
			setBusy(false);
		}
	}

	async function remove(name: string) {
		if (!confirm(`Excluir o agente "${name}"?`)) return;
		try {
			await agentsApi.remove(name);
			if (form.name === name) newAgent();
			await load();
		} catch (e) {
			setError(e instanceof Error ? e.message : String(e));
		}
	}

	return (
		<div className="page agents-page">
			<h1>Agentes</h1>
			<p className="muted">
				Crie agentes (templates) com nome, regras de resposta e modelo. Cada agente é salvo e fica disponível para uso.
			</p>
			{error && <p className="error">{error}</p>}

			<div className="agents-layout">
				{/* Lista */}
				<div className="card agents-list-card">
					<div className="agents-list-head">
						<h3>Seus agentes</h3>
						<button className="link" onClick={newAgent}>
							+ Novo
						</button>
					</div>
					{agents.length === 0 ? (
						<p className="muted">Nenhum agente ainda. Crie o primeiro ao lado.</p>
					) : (
						<ul className="agents-list">
							{agents.map((a) => (
								<li
									key={a.name}
									className={form.name === a.name && editing ? "agent-item active" : "agent-item"}
									onClick={() => edit(a)}
								>
									<div className="agent-item-name">{a.name}</div>
									<div className="agent-item-desc muted">
										{(a.description || a.system_prompt || "").slice(0, 60) || "—"}
									</div>
									<div className="agent-item-tag">{a.model || "modelo padrão"}</div>
								</li>
							))}
						</ul>
					)}
				</div>

				{/* Formulário */}
				<form className="card agents-form" onSubmit={save}>
					<h3>{editing ? `Editar: ${form.name}` : "Novo agente"}</h3>
					<label>
						Nome
						<input
							value={form.name}
							onChange={(e) => set("name", e.target.value)}
							placeholder="ex.: Atendimento, Vendedor, Programador"
							disabled={editing}
							required
						/>
					</label>
					<label>
						Descrição <span className="muted">(opcional)</span>
						<input
							value={form.description || ""}
							onChange={(e) => set("description", e.target.value)}
							placeholder="Resumo curto do que este agente faz"
						/>
					</label>
					<label>
						Regras de resposta <span className="muted">(system prompt)</span>
						<textarea
							value={form.system_prompt}
							onChange={(e) => set("system_prompt", e.target.value)}
							rows={8}
							placeholder="Defina o comportamento: tom, idioma, o que pode e não pode fazer, formato das respostas…"
						/>
					</label>
					<div className="agents-form-row">
						<label>
							Modelo
							<select value={form.model || ""} onChange={(e) => set("model", e.target.value)}>
								<option value="">(modelo padrão)</option>
								{models.map((m) => (
									<option key={m} value={m}>
										{m}
									</option>
								))}
							</select>
						</label>
						<label>
							Temperatura <span className="muted">(0–1)</span>
							<input
								type="number"
								step="0.1"
								min="0"
								max="2"
								value={form.temperature ?? 0.7}
								onChange={(e) => set("temperature", parseFloat(e.target.value) || 0)}
							/>
						</label>
					</div>
					<div className="agents-form-actions">
						<button type="submit" disabled={busy || !form.name.trim()}>
							{busy ? "Salvando…" : "Salvar"}
						</button>
						<button type="button" className="link" onClick={newAgent}>
							Limpar
						</button>
						{editing && (
							<button type="button" className="link danger" onClick={() => remove(form.name)}>
								Excluir
							</button>
						)}
						{msg && <span className="muted">{msg}</span>}
					</div>
				</form>
			</div>
		</div>
	);
}
