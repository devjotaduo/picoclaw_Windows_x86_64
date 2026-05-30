# Comando de Replicação — PicoClaw

> Cole o bloco abaixo para o Claude (ou outro agente) replicar o sistema do zero.
> Acima do bloco fica o **resumo de funcionalidades**; o bloco final é o **comando** pronto.

---

## 📋 Resumo completo das funcionalidades

**PicoClaw** é um assistente de IA pessoal ultraleve, escrito 100% em **Go**, projetado para rodar em hardware de ~US$10 com <10MB de RAM, boot <1s, em arquiteturas RISC-V/ARM/MIPS/x86/LoongArch (binário único, multiplataforma incl. Android).

### 1. Núcleo do Agente (`pkg/agent`)
- Loop de agente com **pipeline** de execução (setup → LLM → execução de tools → finalize).
- **SubTurn**: orquestração de subagentes com controle de concorrência e ciclo de vida.
- **Steering**: injeção de mensagens num loop de agente em execução, entre chamadas de tools.
- **Hooks**: sistema orientado a eventos (observers, interceptors, approval hooks) — montagem por processo.
- **EventBus / Runtime Events**: envelope de eventos, logging centralizado, filtros.
- **Context management**: orçamento de contexto, compactação, integração "seahorse", uso/budget de tokens.
- **Auto-evolução** (`pkg/evolution`): aprende com registros de execução, gera drafts de skills, clusteriza padrões, julga sucesso, aplica melhorias (cold-path runner, profile sync).
- **Memory** (`pkg/memory`): store em JSONL persistente, com migração.
- **Routing** (`pkg/routing`): roteamento de modelos baseado em regras (queries simples → modelos leves p/ economizar custo); classificador + features.
- **Vision pipeline**: envio de imagens/arquivos ao agente, encoding base64 automático p/ LLMs multimodais.
- **Áudio** (`pkg/audio`): ASR (speech-to-text) e TTS (text-to-speech).

### 2. Provedores de LLM (`pkg/providers`) — 30+
OpenAI, Anthropic, Google Gemini, OpenRouter, Zhipu/GLM, DeepSeek, Volcengine, Qwen, Groq, Moonshot/Kimi, Minimax, Mistral, NVIDIA NIM, Cerebras, Novita, Xiaomi MiMo, Ollama, vLLM, LiteLLM, Azure OpenAI, GitHub Copilot (OAuth), Antigravity (OAuth), AWS Bedrock (build tag). Formato `protocolo/modelo` via `model_list`. Adaptadores: openai_compat, anthropic_messages, openai_responses, bedrock, azure, cli, oauth.

### 3. Canais / Apps de mensagem (`pkg/channels`) — 19+
Telegram, Discord, WhatsApp (nativo + bridge), Weixin/WeChat, QQ, Slack (+webhook), Matrix, DingTalk, Feishu/Lark, LINE, WeCom, VK, IRC, OneBot v11, MQTT, MaixCam, Teams (webhook), Pico/Pico Client. Webhooks compartilham um único **Gateway HTTP** (`gateway.host:port`, default `127.0.0.1:18790`).

### 4. Ferramentas embutidas (`pkg/tools`)
- **Arquivos** (`fs`): `read_file`, `read_file_lines`, `write_file`, `edit_file` (com preview de diff), `append_file`, `list_dir`, `load_image`, `send_file`.
- **Shell/Exec**: `shell`/`exec` com políticas de sandbox; sessões de processo (unix/windows).
- **Web**: `web_search` (DuckDuckGo, Gemini Google Search, Baidu, Tavily, Brave, Perplexity, SearXNG, GLM), `web_fetch`.
- **Spawn/Async**: `spawn`, `spawn_status`, `subagent`, `delegate` — tarefas longas e orquestração assíncrona.
- **Cron** (`cron`): lembretes únicos, recorrentes e expressões cron; modos de entrega, gates de comando, persistência.
- **Mensagens**: `message`, `reaction`, `send_tts`.
- **Skills**: `find_skills`, `install_skill` (registries ClawHub + GitHub).
- **MCP** (`pkg/mcp`): Model Context Protocol nativo — stdio/SSE/HTTP, tool discovery; CLI `mcp add/list/test/edit/remove`.
- **Hardware** (`pkg/tools/hardware`): I2C, SPI, Serial (GPIO p/ placas embarcadas).

### 5. CLI (`cmd/picoclaw`)
`onboard`, `agent [-m]`, `gateway`, `status`, `version`, `model`, `auth login/weixin/wecom`, `mcp ...`, `cron add/list/enable/disable/remove`, `skills list/search/install`, `migrate`.

### 6. Web UI / Launcher (`web/`)
- **Launcher** em browser (`http://localhost:18800`, flag `-public`): backend (Go) + frontend (Vite + React 19 + TS + TanStack Router).
- Telas: chat, agent, channels, config (raw + form), credentials, models, logs, launcher setup/login.

### 7. Configuração & Segurança
- `config.json` (dados não sensíveis) + `.security.yml` (segredos, criptografado) — migração de versão 0→1+ automática.
- Sandbox: `allow_read_paths`, `allow_write_paths`, políticas de exec, custom allow/deny patterns, `allow_from`, CORS `allow_origins`.
- Variáveis de ambiente `PICOCLAW_*` (gateway host/port, log level etc.).

### 8. Infra de plataforma
Health checks, heartbeat, identity, isolation, netbind, PID management, updater, tokenizer, credential store, state, session (escopo + JSONL + migração), devices (sources/events), seahorse (compressão de contexto).

### 9. Admin Panel SaaS (`admin-panel/`) — add-on
Painel externo (Node + Express + better-sqlite3 + React 19/Vite) para gerenciar clientes do SaaS: cada cliente recebe token de acesso e `config.json` pronto com OpenRouter pré-configurado. Backend `:4000`, frontend `:5174`.

### Build & Deploy
Go 1.25+, Node 22+/pnpm; `make build`, `make build-launcher`, `make build-all`, builds cross-arch; Docker Compose (perfil launcher); GoReleaser; golangci-lint.

---

## 🤖 COMANDO (cole isto para o agente replicar)

```
Construa do zero um assistente de IA pessoal ultraleve em Go chamado "PicoClaw",
otimizado para hardware de baixo custo (<10MB RAM, boot <1s, binário único
multiplataforma: x86/ARM/RISC-V/MIPS/Android). Replique TODAS as funcionalidades
abaixo, com arquitetura modular em pacotes:

1. NÚCLEO DO AGENTE (pkg/agent): loop de agente com pipeline
   (setup→LLM→tools→finalize); SubTurn (orquestração de subagentes c/ concorrência
   e ciclo de vida); Steering (injetar mensagens no loop entre tool-calls); sistema
   de Hooks orientado a eventos (observers, interceptors, approval hooks); EventBus
   + runtime events com logging centralizado e filtros; gerenciamento de contexto
   com orçamento/compactação de tokens; auto-evolução (aprende de execuções, gera
   drafts de skills, clusteriza padrões, julga sucesso, aplica); memória persistente
   em JSONL; vision pipeline (imagens/arquivos → base64 p/ LLM multimodal); ASR+TTS.

2. PROVEDORES LLM (pkg/providers): suporte a 30+ provedores via formato
   "protocolo/modelo" em model_list — OpenAI, Anthropic, Gemini, OpenRouter, Zhipu,
   DeepSeek, Qwen, Groq, Moonshot/Kimi, Minimax, Mistral, NVIDIA, Cerebras, Ollama,
   vLLM, LiteLLM, Azure, GitHub Copilot (OAuth), AWS Bedrock, etc. Adaptadores
   openai-compatible, anthropic-messages, openai-responses, oauth, cli.
   Roteamento de modelos por regras (queries simples → modelos leves).

3. CANAIS (pkg/channels): integração com 19+ apps de mensagem — Telegram, Discord,
   WhatsApp, WeChat/Weixin, QQ, Slack, Matrix, DingTalk, Feishu, LINE, WeCom, VK,
   IRC, OneBot, MQTT, Teams, MaixCam. Webhooks compartilham um Gateway HTTP único.

4. FERRAMENTAS EMBUTIDAS (pkg/tools): arquivos (read/write/edit c/ diff preview,
   append, list_dir, load_image, send_file); shell/exec com sandbox; web_search
   (DuckDuckGo, Tavily, Brave, Perplexity, SearXNG, Baidu, Gemini, GLM) + web_fetch;
   spawn/subagent/delegate p/ tarefas assíncronas; cron (lembretes únicos/recorrentes/
   expressões cron); message/reaction/send_tts; skills (find/install via ClawHub+GitHub);
   MCP nativo (stdio/SSE/HTTP, tool discovery); hardware I2C/SPI/Serial p/ GPIO.

5. CLI (cmd/picoclaw): onboard, agent [-m], gateway, status, version, model,
   auth login, mcp add/list/test/edit/remove, cron add/list/enable/disable/remove,
   skills list/search/install, migrate.

6. WEB UI / LAUNCHER (web/): launcher em browser na porta 18800 (flag -public),
   backend Go + frontend Vite/React 19/TypeScript/TanStack Router, com telas de
   chat, agent, channels, config, credentials, models e logs.

7. CONFIG & SEGURANÇA: config.json (não sensível) + .security.yml (segredos
   criptografados) com migração de versão automática; sandbox de paths read/write,
   políticas de exec, allow/deny patterns, CORS; variáveis de ambiente PICOCLAW_*.

8. INFRA: health checks, heartbeat, identity, isolation, netbind, PID, updater,
   tokenizer, credential store, sessões (JSONL persistente + migração), devices,
   compressão de contexto.

9. ADMIN PANEL SaaS (opcional, admin-panel/): painel externo Node+Express+
   better-sqlite3+React p/ gerenciar clientes — cada cliente recebe token e
   config.json pronto com OpenRouter.

Build com Go 1.25+, Makefile com targets cross-arch (build, build-launcher,
build-all), Docker Compose e GoReleaser. Mantenha o código pequeno, idiomático e
legível. Comece pela estrutura de pacotes e pelo loop do agente + um provedor
(OpenAI) + um canal (Telegram) + ferramentas de arquivo/shell, depois expanda.
```

---

## 🤖 COMMAND (English version)

```
Build from scratch an ultra-lightweight personal AI assistant in Go called
"PicoClaw", optimized for low-cost hardware (<10MB RAM, <1s boot, single
cross-platform binary: x86/ARM/RISC-V/MIPS/Android). Replicate ALL of the
features below, using a modular package architecture:

1. AGENT CORE (pkg/agent): agent loop with an execution pipeline
   (setup→LLM→tools→finalize); SubTurn (subagent orchestration with concurrency
   and lifecycle control); Steering (inject messages into the running loop between
   tool calls); event-driven Hook system (observers, interceptors, approval hooks);
   EventBus + runtime events with centralized logging and filters; context
   management with token budget/compaction; self-evolution (learns from runs,
   generates skill drafts, clusters patterns, judges success, applies improvements);
   persistent JSONL memory; vision pipeline (images/files → base64 for multimodal
   LLMs); ASR + TTS.

2. LLM PROVIDERS (pkg/providers): support 30+ providers via a "protocol/model"
   format in model_list — OpenAI, Anthropic, Gemini, OpenRouter, Zhipu, DeepSeek,
   Qwen, Groq, Moonshot/Kimi, Minimax, Mistral, NVIDIA, Cerebras, Ollama, vLLM,
   LiteLLM, Azure, GitHub Copilot (OAuth), AWS Bedrock, etc. Adapters for
   openai-compatible, anthropic-messages, openai-responses, oauth, cli.
   Rule-based model routing (simple queries → lightweight models).

3. CHANNELS (pkg/channels): integrate 19+ messaging apps — Telegram, Discord,
   WhatsApp, WeChat/Weixin, QQ, Slack, Matrix, DingTalk, Feishu, LINE, WeCom, VK,
   IRC, OneBot, MQTT, Teams, MaixCam. Webhook channels share a single HTTP Gateway.

4. BUILT-IN TOOLS (pkg/tools): files (read/write/edit with diff preview, append,
   list_dir, load_image, send_file); sandboxed shell/exec; web_search (DuckDuckGo,
   Tavily, Brave, Perplexity, SearXNG, Baidu, Gemini, GLM) + web_fetch;
   spawn/subagent/delegate for async tasks; cron (one-shot/recurring/cron-expr
   reminders); message/reaction/send_tts; skills (find/install via ClawHub+GitHub);
   native MCP (stdio/SSE/HTTP, tool discovery); hardware I2C/SPI/Serial for GPIO.

5. CLI (cmd/picoclaw): onboard, agent [-m], gateway, status, version, model,
   auth login, mcp add/list/test/edit/remove, cron add/list/enable/disable/remove,
   skills list/search/install, migrate.

6. WEB UI / LAUNCHER (web/): browser launcher on port 18800 (-public flag),
   Go backend + Vite/React 19/TypeScript/TanStack Router frontend, with screens for
   chat, agent, channels, config, credentials, models, and logs.

7. CONFIG & SECURITY: config.json (non-sensitive) + .security.yml (encrypted
   secrets) with automatic version migration; sandbox for read/write paths, exec
   policies, allow/deny patterns, CORS; PICOCLAW_* environment variables.

8. INFRA: health checks, heartbeat, identity, isolation, netbind, PID, updater,
   tokenizer, credential store, sessions (persistent JSONL + migration), devices,
   context compression.

9. SaaS ADMIN PANEL (optional, admin-panel/): external Node+Express+better-sqlite3+
   React panel to manage clients — each client gets an access token and a ready-made
   config.json pre-configured with OpenRouter.

Build with Go 1.25+, a Makefile with cross-arch targets (build, build-launcher,
build-all), Docker Compose, and GoReleaser. Keep the code small, idiomatic, and
readable. Start with the package structure and the agent loop + one provider
(OpenAI) + one channel (Telegram) + file/shell tools, then expand.
```
