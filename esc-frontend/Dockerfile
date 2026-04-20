# ── Build stage ──────────────────────────────────────────────────────────────
FROM node:22-alpine AS build

RUN corepack enable && corepack prepare pnpm@10.10.0 --activate

WORKDIR /app

# Install dependencies first (layer cache)
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY client/package.json client/

RUN pnpm install --frozen-lockfile

# Copy source and build client
COPY client/ client/

RUN pnpm --filter @esc/client build

# ── Production stage ─────────────────────────────────────────────────────────
FROM nginx:1.27-alpine

RUN apk add --no-cache curl

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/client/dist /usr/share/nginx/html

EXPOSE 3001
