# Deploy em produção

Arquitetura: **frontends estáticos na Vercel** + **backends de processo longo em Docker (Railway/Fly)**.
A Vercel é serverless e não roda o agente Go 24/7, SQLite ou WhatsApp — por isso os backends ficam no Railway.

```
Vercel (estático, CDN)                 Railway / Fly (Docker, volume /data)
┌──────────────────────────┐           ┌────────────────────────────────────┐
│ launcher  ui-jotduo       │ ──/api──▶ │ picoclaw web  (Go)        :18800     │
│ admin     web-jotduo      │ ──/api──▶ │ admin-panel API (Node)    :4000      │
└──────────────────────────┘           └────────────────────────────────────┘
        VITE_API_BASE                        PICOCLAW_CORS_ORIGIN / CORS_ORIGIN
```

## Estado atual

| Peça | Onde | Status |
|------|------|--------|
| Launcher UI (`web/ui`) | Vercel projeto `ui` → https://ui-jotduo.vercel.app | ✅ deployed (prod) |
| Admin UI (`admin-panel/web`) | Vercel projeto `web` → https://web-jotduo.vercel.app | ✅ deployed (prod) |
| Launcher backend (`Dockerfile`) | Railway | ⏳ falta você criar o serviço |
| Admin backend (`admin-panel/server/Dockerfile`) | Railway | ⏳ falta você criar o serviço |

> Os frontends estão no ar mas ainda **não conversam com nenhum backend** até você concluir os passos 1–3 abaixo.

---

## 1. Tirar a proteção da Vercel (deixar público)

Hoje as URLs retornam **401** por causa do *Deployment Protection*. Para deixar público:

Dashboard Vercel → projeto `ui` → **Settings → Deployment Protection → Vercel Authentication → Disabled → Save**.
Repita no projeto `web`.

(Ou via API, com um token criado em vercel.com/account/tokens:
`curl -X PATCH "https://api.vercel.com/v9/projects/ui?teamId=<team>" -H "Authorization: Bearer <TOKEN>" -H "Content-Type: application/json" -d '{"ssoProtection":null}'`.)

---

## 2. Subir os backends no Railway

Pré-requisito: conta Railway + repo no GitHub (`devjotaduo/picoclaw_Windows_x86_64`).

### 2a. Serviço `picoclaw` (launcher Go)
1. Railway → **New Project → Deploy from GitHub repo** → selecione o repo.
2. O `railway.json` na raiz já aponta para o `Dockerfile` da raiz. Railway detecta sozinho.
3. **Settings → Volumes**: monte um volume em `/data` (guarda senha, credenciais e config entre deploys).
4. **Variables**:
   - `PICOCLAW_SECRET` = string aleatória longa (assina o cookie de sessão; sem isso a sessão cai a cada restart).
   - `PICOCLAW_CORS_ORIGIN` = `https://ui-jotduo.vercel.app` (origem exata do frontend; habilita CORS + cookie `SameSite=None;Secure`).
   - `PORT` é injetado pelo Railway automaticamente.
5. Deploy. Anote a URL pública (ex.: `https://picoclaw-production.up.railway.app`).

### 2b. Serviço `admin` (Node)
1. No mesmo projeto: **New Service → GitHub repo (mesmo repo)**.
2. **Settings → Root Directory** = `admin-panel/server` (usa o Dockerfile de lá).
3. **Volume** em `/data`.
4. **Variables**:
   - `ADMIN_PASSWORD` = senha do painel admin.
   - `COOKIE_SECRET` = string aleatória longa.
   - `OPENROUTER_API_KEY` = sua chave (vai nos config.json provisionados).
   - `CORS_ORIGIN` = `https://web-jotduo.vercel.app`.
   - `DEFAULT_MODEL` (opcional) = `openrouter/openai/gpt-4o-mini`.
5. Deploy. Anote a URL (ex.: `https://admin-production.up.railway.app`).

---

## 3. Ligar os frontends aos backends (`VITE_API_BASE`)

`VITE_API_BASE` é **build-time** — depois de setar, precisa **redeploy**.

```bash
# launcher
cd web/ui
vercel env add VITE_API_BASE production    # cole a URL do serviço picoclaw no Railway
vercel deploy --prod --yes

# admin
cd ../../admin-panel/web
vercel env add VITE_API_BASE production    # cole a URL do serviço admin no Railway
vercel deploy --prod --yes
```

Se o domínio público da Vercel mudar, atualize `PICOCLAW_CORS_ORIGIN` / `CORS_ORIGIN` no Railway de acordo.

---

## Variáveis de ambiente (resumo)

### Launcher Go (Railway)
| Var | Exemplo | Função |
|-----|---------|--------|
| `PICOCLAW_CORS_ORIGIN` | `https://ui-jotduo.vercel.app` | CORS + cookie cross-site |
| `PICOCLAW_SECRET` | (aleatório) | assina o cookie de sessão (persistente) |
| `PORT` | (auto) | porta de bind |

### Admin Node (Railway)
| Var | Exemplo | Função |
|-----|---------|--------|
| `CORS_ORIGIN` | `https://web-jotduo.vercel.app` | CORS + cookie cross-site |
| `ADMIN_PASSWORD` | (segredo) | login do painel |
| `COOKIE_SECRET` | (aleatório) | assina o cookie |
| `OPENROUTER_API_KEY` | `sk-or-...` | chave nos configs provisionados |
| `DEFAULT_MODEL` | `openrouter/openai/gpt-4o-mini` | modelo padrão |

### Frontends (Vercel)
| Var | Valor | Função |
|-----|-------|--------|
| `VITE_API_BASE` | URL do backend Railway | origem da API (build-time) |

---

## Build local de teste do container (opcional, precisa Docker)

```bash
# launcher
docker build -t picoclaw .
docker run -p 18800:18800 -v picoclaw-data:/data \
  -e PICOCLAW_SECRET=dev -e PICOCLAW_CORS_ORIGIN=http://localhost:5173 picoclaw

# admin
docker build -t picoclaw-admin admin-panel/server
docker run -p 4000:4000 -v admin-data:/data \
  -e ADMIN_PASSWORD=admin -e COOKIE_SECRET=dev -e CORS_ORIGIN=http://localhost:5174 picoclaw-admin
```
