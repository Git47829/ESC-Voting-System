# ESC Frontend Monorepo (React + Express + TypeScript)

A full-stack monorepo application for Eurovision voting with React client and Express server.

## Tech Stack

- **Client**: React 18, TypeScript 5.8, Vite 5, Tailwind CSS 3, React Router 6
- **Server**: Express 4, TypeScript 5.8, Node.js 22
- **Package Manager**: pnpm 10.10.0
- **Build Tool**: Vite 5
- **Additional Tools**: OpenTelemetry (tracing/monitoring), Express Session, CORS

## Project Structure

```
esc-frontend/
├── client/                 # React frontend application
│   ├── src/
│   │   ├── components/    # Reusable UI and feature components
│   │   ├── pages/         # Page components (Vote, Results, Admin, etc.)
│   │   ├── services/      # API client services
│   │   ├── context/       # React context (Auth, Flash messages, Cookies)
│   │   ├── hooks/         # Custom React hooks
│   │   ├── types/         # TypeScript type definitions
│   │   ├── utils/         # Utility functions
│   │   ├── App.tsx        # Main app router
│   │   └── main.tsx       # Entry point
│   ├── vite.config.ts     # Vite configuration with dev server proxy
│   └── package.json       # Client dependencies & scripts
│
├── server/                # Express backend
│   ├── src/
│   │   ├── routes/        # API endpoints
│   │   ├── services/      # Business logic layer (Auth, Voting, Results)
│   │   ├── middleware/    # Auth, CSRF, error handling
│   │   ├── mock/          # Mock data services for development
│   │   ├── config.ts      # Configuration management
│   │   ├── index.ts       # Server entry point
│   │   └── ...
│   ├── .env.example       # Environment variables template
│   └── package.json       # Server dependencies & scripts
│
├── pnpm-workspace.yaml    # Monorepo workspace configuration
└── package.json           # Root workspace scripts
```

## Features

- **Public Voting**: Vote for Eurovision songs with points
- **Jury Voting**: Jury members can cast votes with official jury point values (1, 2, 3, 4, 5, 6, 7, 8, 10, 12)
- **Admin Dashboard**: Manage contest state, add/edit entries
- **Results Page**: Real-time vote aggregation and rankings
- **Stats Dashboard**: Detailed voting statistics and analytics
- **Now Playing**: Current song information display
- **Authentication**: Role-based access control (Admin, Jury, Public)
  - Token-based authentication
  - Email + OTP verification
- **Cookie Consent**: User-controlled cookie preferences
- **OpenTelemetry Tracing**: Request monitoring and observability

## Prerequisites

- **Node.js**: 20+ (recommend 22)
- **pnpm**: 10.10.0+ (npm/yarn not recommended for workspaces)

## Getting Started

### Development Setup

1. Install dependencies:
   ```bash
   cd esc-frontend
   pnpm install
   ```

2. Set up environment (optional - mock mode is default):
   ```bash
   cp server/.env.example server/.env
   ```

3. Start development servers:
   ```bash
   pnpm dev
   ```

   This starts both client and server in parallel:
   - **Client**: http://localhost:5173 (React dev server with Vite)
   - **Server**: http://localhost:3001 (Express backend)

   The client dev server has a proxy configured to forward `/api` requests to the server.

### Scripts

- `pnpm dev` - Start development servers (client + server)
- `pnpm build` - Build both client and server for production
- `pnpm lint` - Run TypeScript type checking on both client and server

## Environment Configuration

### Development (Default - Mock Mode)

Mock mode is enabled by default when `NODE_ENV=development`. This provides:
- Pre-populated countries and songs
- Mock authentication (token-based)
- In-memory vote storage
- No external service dependencies

**Mock Credentials**:
- Admin token: `admin-token`
- Jury token: `jury-token`

### Production (Real Backend Mode)

Create `server/.env` to enable real backend services:

```env
# Server Configuration
NODE_ENV=production
PORT=3001
SESSION_SECRET=your-secure-secret-here

# Backend Services
USE_MOCK=false
API_BASE_URL=http://db-crud-api:8000
API_TIMEOUT=10000
ESC_CONVERTER_URL=http://public-vote-converter:8090
EUROSTATS_URL=http://eurostats:8880

# Contest Settings
TOTAL_VOTE_POINTS=20
```

**Environment Variables**:
- `NODE_ENV` - Set to `production` for real backend, `development` for mock mode
- `USE_MOCK` - Force mock mode (`true`/`false`) - overrides NODE_ENV default
- `PORT` - Server port (default: 3001)
- `SESSION_SECRET` - Session encryption secret (required in production)
- `API_BASE_URL` - Upstream database API base URL
- `API_TIMEOUT` - Request timeout in milliseconds
- `ESC_CONVERTER_URL` - Vote converter service URL
- `EUROSTATS_URL` - Statistics aggregation service URL
- `TOTAL_VOTE_POINTS` - Total points available for voting (default: 20)

## API Endpoints

**Authentication**:
- `POST /api/login` - Login with token
- `POST /api/auth/login` - Email login (sends OTP)
- `POST /api/auth/verify` - Verify email OTP
- `POST /api/logout` - Logout
- `GET /api/session` - Get current session info

**Contest/Songs**:
- `GET /api/songs` - Fetch all songs and countries
- `GET /api/contest/state` - Get contest status

**Voting**:
- `GET /api/votes` - Fetch vote results
- `POST /api/votes/public` - Cast public vote
- `POST /api/votes/jury` - Cast jury vote
- `GET /api/votes/jury/state` - Get jury vote state

**Results**:
- `GET /api/results` - Aggregate vote results
- `GET /api/results/stats` - Detailed voting statistics

**Health**:
- `GET /health` - Server health check

## Client Routes

- `/` - Public voting page
- `/now` - Now playing / current song
- `/results` - Vote results and rankings
- `/stats` - Detailed statistics
- `/cookies` - Cookie consent settings
- `/login` - Authentication
- `/admin` - Admin dashboard (requires admin role)
- `/jury` - Jury voting page (requires jury role)

## Building for Production

```bash
pnpm build
```

This generates:
- `client/dist/` - React app bundle (Vite)
- `server/dist/` - Compiled TypeScript (Node.js)

Both are included in the Docker image via `Dockerfile`.

## Type Checking & Linting

```bash
pnpm lint
```

Runs TypeScript type checking on both client and server (no code modifications, just validation).

## Troubleshooting

**Port conflicts**: If ports 5173 or 3001 are in use, modify `client/vite.config.ts` and `server/.env`.

**CORS issues**: In development, CORS is automatically enabled for `http://localhost:5173`. In production, adjust CORS settings in `server/src/index.ts`.

**Session issues**: Ensure `SESSION_SECRET` is set. In development with mock mode, this is handled automatically.

**Real backend not connecting**: Verify upstream service URLs in `.env` match your deployment, and services are accessible from the server.

## Architecture Notes

The server uses a **service factory pattern** with dependency injection:
- Requests flow through middleware (auth, CSRF) → routes → services → data layer
- Services abstract business logic from HTTP handling
- Mock and production implementations share the same interface
- This enables easy switching between mock and real backend modes

OpenTelemetry tracing provides production-grade observability for request flows and errors.
