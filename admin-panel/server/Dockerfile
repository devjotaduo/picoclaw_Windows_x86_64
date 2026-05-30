# PicoClaw admin panel API (Node + Express + node:sqlite) — production image.
# Build context: admin-panel/server.  Mount a persistent volume at /data so the
# tenants database and provisioned workspaces survive redeploys.
FROM node:24-alpine
RUN adduser -D -u 10001 app
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV NODE_ENV=production \
    PORT=4000 \
    DATA_DIR=/data \
    WORKSPACES_ROOT=/data/workspaces
RUN mkdir -p /data && chown -R app /data /app
USER app
EXPOSE 4000
CMD ["npm", "run", "start"]
