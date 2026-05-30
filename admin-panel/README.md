# PicoClaw Admin Panel (MVP)

Painel SaaS para cadastrar e gerenciar o **acesso dos clientes**. Cada cliente
recebe um PicoClaw isolado: workspace próprio, `access_token` (`pc_…`) próprio e
um `config.json` provisionado para download.

Dois processos:

| Parte    | Stack                                   | Porta |
|----------|-----------------------------------------|-------|
| `server/`| Node + Express + `node:sqlite` (TS/tsx) | 4000  |
| `web/`   | Vite + React 19 + react-router-dom      | 5174  |

O frontend (`5174`) faz proxy de `/api` para o backend (`4000`), então tudo é
mesma-origem em dev.

## Rodar (dev)

```bash
# 1) Backend
cd admin-panel/server
cp .env.example .env          # ajuste ADMIN_PASSWORD e OPENROUTER_API_KEY
npm install
npm run dev                   # http://127.0.0.1:4000

# 2) Frontend (outro terminal)
cd admin-panel/web
pnpm install                  # (ou npm install)
pnpm dev                      # http://127.0.0.1:5174
```

Abra `http://127.0.0.1:5174`, entre com a senha de `ADMIN_PASSWORD` e cadastre
clientes.

## Segurança

- Auth por **senha única** (`ADMIN_PASSWORD`) + cookie de sessão HTTP-only
  assinado por HMAC (`COOKIE_SECRET`).
- A chave **OpenRouter** fica só no servidor (`OPENROUTER_API_KEY`); ela só
  aparece no `config.json` baixado pelo admin, nunca em uma superfície
  voltada ao cliente.

## API

| Método | Rota                                  | Função                          |
|--------|---------------------------------------|---------------------------------|
| POST   | `/api/auth/login`                     | login (senha)                   |
| POST   | `/api/auth/logout`                    | logout                          |
| GET    | `/api/auth/me`                        | status de sessão                |
| GET    | `/api/health`                         | healthcheck                     |
| GET    | `/api/clients`                        | listar clientes                 |
| POST   | `/api/clients`                        | criar cliente (provisiona)      |
| DELETE | `/api/clients/:id`                    | remover cliente                 |
| POST   | `/api/clients/:id/regenerate-token`   | gerar novo `access_token`       |
| POST   | `/api/clients/:id/status`             | ativar / suspender              |
| GET    | `/api/clients/:id/provision`          | baixar `config.json` do cliente |

## Provisionamento

Ao criar um cliente, o backend gera `id`, `slug`, `access_token` (`pc_…`) e cria
um diretório de workspace isolado em `WORKSPACES_ROOT/<slug>`. O endpoint
`/provision` monta um `config.json` PicoClaw com esse workspace, `model_list[0]`
apontando para OpenRouter (chave do servidor) e `agents.defaults.model_name`.
