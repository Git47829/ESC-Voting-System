# ESC Voting System — Flask Frontend (`frontend/`)

Legacy Flask + Jinja frontend for the ESC voting system.

> This frontend is still implemented in this folder, but the current `docker-compose.yaml` frontend service builds `esc-frontend/`.

## Stack

- Python 3.12
- Flask 3
- Jinja2 templates + Tailwind CDN
- Gunicorn (container runtime)
- OpenTelemetry + Prometheus metrics (`telemetry.py`)

## Run locally

```bash
cd frontend
python3 -m venv .venv
. .venv/bin/activate
pip install -r src/requirements.txt
python src/main.py
```

App defaults to `http://localhost:5000`.

## Environment variables

- `FLASK_PORT` (default `5000`)
- `FLASK_DEBUG` (`true`/`false`, default `false`)
- `FLASK_SECRET_KEY`
- `API_BASE_URL` (default `http://db-crud-api:8000`)
- `API_TIMEOUT` (seconds, default `10`)
- `ESC_CONVERTER_URL` (default `http://public-vote-converter:8090`)
- `EUROSTATS_URL` (default `http://eurostats:8880`)
- `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` (default `http://otel-collector:4318`)
- `OTEL_EXPORTER_OTLP_GRPC_ENDPOINT` (default `http://otel-collector:4317`)
- `PROMETHEUS_MULTIPROC_DIR` (needed for Gunicorn multiprocess metrics)

## Architecture

- `src/main.py` creates the app, initializes telemetry, and registers blueprints.
- Route modules:
  - `routes/public.py`
  - `routes/auth.py`
  - `routes/admin.py`
  - `routes/jury.py`
- Shared backend HTTP helpers in `src/api_client.py`.
- Auth/session guards and helpers in `src/utils.py`.

## Runtime behavior

### Auth

Login is a **2-step flow**:
1. `POST /auth/login` with email/password/role
2. `POST /auth/verify` with 6-digit code

On success, Flask session stores `role`, `email`, and `token` (token uses the submitted password value in current implementation).

### Public voting

- Public votes are submitted to `POST /vote/submit`.
- Consent cookie `esc_cookie_consent` must include `preferences.essential = true`.
- Vote budget is tracked through `vote_state` cookie state from the backend.

### Live pages

- `/results`: polls `/api/results` every 10s.
- `/now`: polls `/api/contest/current` every 5s and reloads when song index changes.
- `/stats`: frontend JS opens `wss://<host>/eurostats/ws/stats` for live chart updates.

## Flask routes

- Public pages: `/`, `/results`, `/stats`, `/cookies`, `/now`, `/login`, `/logout`
- Public JSON: `/api/results`, `/api/contest/current`
- Vote submit: `POST /vote/submit`
- Admin actions: `/admin/*`
- Jury actions: `/jury`, `POST /jury/submit`
- Ops endpoints: `/health`, `/metrics`

## Integration points

Backend dependencies used by this frontend:

- CRUD API (`API_BASE_URL`) for songs, auth, votes, admin, jury
- Public Vote Converter (`ESC_CONVERTER_URL`) for ESC public points in results
- EuroStats (`EUROSTATS_URL`) reset endpoint on vote reset; live stats websocket is consumed from `/eurostats/ws/stats` through reverse proxy routing
