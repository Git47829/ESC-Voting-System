# ESC Frontend Monorepo (React + Express + TypeScript)

## Setup

```bash
cd /Users/rodimemboyrazli/IdeaProjects/ESC-Voting-System/esc-frontend
pnpm install
pnpm dev
```

- Client: `http://localhost:5173`
- Server: `http://localhost:3001`

## Build

```bash
pnpm build
```

## Mock credentials

- Admin token: `admin-token`
- Jury token: `jury-token`

## Real backend mode

Create `server/.env`:

```env
NODE_ENV=production
USE_MOCK=false
API_BASE_URL=http://db-crud-api:8000
API_TIMEOUT=10000
ESC_CONVERTER_URL=http://public-vote-converter:8090
EUROSTATS_URL=http://eurostats:8880
PORT=3001
SESSION_SECRET=change-me-in-production
TOTAL_VOTE_POINTS=20
```

In development, mock mode is active by default (`NODE_ENV=development`).

