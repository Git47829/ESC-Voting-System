# ESC Frontend Monorepo (`esc-frontend/`)

Current frontend implementation: React SPA (`client/`) + Express BFF (`server/`) in a pnpm workspace.

## Workspace layout

- `client/` — React 18 + TypeScript + Vite
- `server/` — Express + TypeScript API/BFF + session/auth handling
- `pnpm-workspace.yaml` links both packages

## Prerequisites

- Node.js 22+
- pnpm 10.10.0

## Setup & run

```bash
cd esc-frontend
pnpm install
pnpm dev
```

Development endpoints:

- Client: `http://localhost:5173`
- Server API: `http://localhost:3001`

## Scripts

Root scripts:

- `pnpm dev` — run `client` and `server` in parallel
- `pnpm build` — build both workspaces
- `pnpm lint` — TypeScript no-emit checks in both workspaces

Package scripts:

- `client`: `dev`, `build`, `preview`, `lint`
- `server`: `dev`, `build`, `start`, `lint`

## Runtime architecture

### Client

- Routes: `/`, `/now`, `/results`, `/stats`, `/cookies`, `/login`, `/admin`, `/jury`
- `ProtectedRoute` enforces role access for `/admin` and `/jury`
- Polling:
  - results every 10s (`useResultsPoll`)
  - contest current song every 5s (`useContestPoll`)
  - voting status every 5s (`useVotingStatus`)
- Cookie consent is stored client-side (`esc_cookie_consent`)

### Server (BFF)

- Exposes app health at `GET /health`
- Exposes frontend API under `GET/POST /api/*`
- Uses Redis-backed `express-session`
- In development, enables CORS for `http://localhost:5173` and CSRF middleware (`/api/csrf-token`)
- Role checks via `requireRole("admin" | "jury")`

## Auth behavior

Implemented API supports both flows:

1. Token auth (`POST /api/login`)
2. Two-step auth (`POST /api/auth/login` then `POST /api/auth/verify`)

Current UI uses the two-step email/code flow from `LoginPage`.

## Mock vs real backend

`USE_MOCK` controls data source:

- Mock mode: in-memory data (`server/src/mock/*`)
- Real mode: proxies to backend services

Default behavior: if `USE_MOCK` is unset, mock mode is on in `NODE_ENV=development`.

## Environment variables (server)

- `NODE_ENV` (default `development`)
- `USE_MOCK`
- `API_BASE_URL` (default `http://db-crud-api:8000`)
- `API_TIMEOUT` (ms, default `10000`)
- `ESC_CONVERTER_URL` (default `http://public-vote-converter:8090`)
- `EUROSTATS_URL` (default `http://eurostats:8880`)
- `REDIS_URL` (default `redis://redis:6379`)
- `PORT` (default `3001`)
- `SESSION_SECRET`
- `TOTAL_VOTE_POINTS` (default `20`)

## Integration points

Server routes integrate with:

- CRUD API (`API_BASE_URL`) for songs, auth, voting, admin, jury
- Public Vote Converter (`ESC_CONVERTER_URL`) for combined results (`/api/results`)
- EuroStats (`EUROSTATS_URL`) for stats endpoint (`/api/stats`)

The Stats page opens WebSocket `ws(s)://<host>/eurostats/ws/stats` directly from the browser.

## Docker

`esc-frontend/Dockerfile` builds both workspaces, serves compiled client via Express in production, and starts `node server/dist/index.js` on port `3001`.
