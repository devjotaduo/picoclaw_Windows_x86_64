# PicoClaw admin panel API (Node + Express + node:sqlite) — production image.
# Build context: admin-panel/server.  Mount a persistent volume at /data so the
# tenants database and provisioned workspaces survive redeploys.
FROM node:24-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV NODE_ENV=production \
    PORT=4000 \
    DATA_DIR=/data \
    WORKSPACES_ROOT=/data/workspaces
# Runs as root so a root-owned Railway/Fly volume mounted at /data is writable.
RUN mkdir -p /data
EXPOSE 4000
CMD ["npm", "run", "start"]
