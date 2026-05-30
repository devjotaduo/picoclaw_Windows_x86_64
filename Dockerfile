# PicoClaw web launcher (Go agent + embedded React UI) — production image.
# Build context: repository root.  Deploy target: Railway / Fly / any Docker host.
# Mount a persistent volume at /data so the password, credentials and config
# survive redeploys.

# syntax=docker/dockerfile:1

# --- Stage 1: build the React launcher UI (embedded into the Go binary) ---
FROM node:22-alpine AS ui
RUN corepack enable
WORKDIR /app/web/ui
COPY web/ui/package.json web/ui/pnpm-lock.yaml web/ui/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ui ./
# Non-VERCEL build → emits to ../server/dist, which Go embeds.
RUN pnpm build

# --- Stage 2: compile the Go binary (embeds web/server/dist) ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
COPY --from=ui /app/web/server/dist ./web/server/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/picoclaw ./cmd/picoclaw

# --- Stage 3: runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/picoclaw /usr/local/bin/picoclaw
COPY deploy/config.template.json /app/config.template.json
COPY deploy/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh && mkdir -p /data && chown app /data
USER app
ENV PICOCLAW_DATA=/data PORT=18800
EXPOSE 18800
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
