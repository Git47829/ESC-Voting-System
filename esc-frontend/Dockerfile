# ── Build stage ──────────────────────────────────────────────────────────────
FROM node:22-alpine AS build

RUN corepack enable && corepack prepare pnpm@10.10.0 --activate

WORKDIR /app

# Install dependencies first (layer cache)
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY client/package.json client/
COPY server/package.json server/

RUN pnpm install --frozen-lockfile

# Copy source and build
COPY client/ client/
COPY server/ server/

RUN pnpm -r build

# ── Production stage ─────────────────────────────────────────────────────────
FROM node:22-alpine

RUN corepack enable && corepack prepare pnpm@10.10.0 --activate

WORKDIR /app

# Copy built artifacts
COPY --from=build /app/client/dist client/dist
COPY --from=build /app/server/dist server/dist

# Install production deps for server only
COPY server/package.json server/
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

RUN pnpm install --prod --frozen-lockfile --filter @esc/server

EXPOSE 3001

CMD ["node", "server/dist/index.js"]
