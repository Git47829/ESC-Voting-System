# ESC Frontend (`esc-frontend/`)

Frontend workspace for the React SPA (`client/`).

## Workspace layout

- `client/` — React 18 + TypeScript + Vite

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
- CRUD API (direct): `http://localhost:8000`

## Scripts

Root scripts:

- `pnpm dev` — run the React client in dev mode
- `pnpm build` — build the client
- `pnpm lint` — TypeScript no-emit check for the client

Package scripts:

- `client`: `dev`, `build`, `preview`, `lint`

## Client runtime architecture

- Routes: `/`, `/now`, `/results`, `/stats`, `/cookies`, `/login`, `/admin`, `/jury`
- `ProtectedRoute` enforces role access for `/admin` and `/jury`
- Polling:
  - results every 10s (`useResultsPoll`)
  - contest current song every 5s (`useContestPoll`)
  - voting status every 5s (`useVotingStatus`)
- Cookie consent is stored client-side (`esc_cookie_consent`)

## Client API integration (direct Go services)

The React client talks directly to backend services:

- CRUD-DB-API (`/crud-api` by default): auth, songs, votes, countries, contest, admin, jury, results
- EuroStats WebSocket (`/eurostats/ws/stats` by default): live stats

Auth uses HttpOnly JWT cookies (`credentials: include`) with:

- `POST /auth/login`
- `POST /auth/verify`
- `GET /auth/me`
- `POST /auth/refresh`
- `POST /auth/logout`

## Client environment variables

Create `client/.env` (see `client/.env.example`):

- `VITE_CRUD_API_BASE_URL` (default `/crud-api`)
- `VITE_EUROSTATS_WS_URL` (optional explicit WS URL)

## Vite dev proxy

`client/vite.config.ts` proxies:

- `/crud-api` → `http://localhost:8000`
- `/eurostats` (WS enabled) → `http://localhost:8880`

## Docker

`esc-frontend/Dockerfile` builds the client and serves static assets via Nginx on port `3001`.
