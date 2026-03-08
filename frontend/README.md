# ESC Voting System — Frontend

Flask + Jinja2 + Tailwind CSS (CDN) frontend for the Eurovision Song Contest Voting System. Served in production by Gunicorn with 2 workers and full OpenTelemetry instrumentation.

## Design Tokens

| Token       | Value     |
|-------------|-----------|
| `--yellow`  | `#ffde00` |
| `--black`   | `#0a0a0a` |
| `--surface` | `#1a1a1a` |
| `--text`    | `#e8e8e8` |
| `--muted`   | `#666`    |

**Fonts:** Syne (headings) + DM Mono (body/labels) — loaded from Google Fonts.

## Pages

| Route          | Page            | Auth         | Description                                                   |
|----------------|-----------------|--------------|---------------------------------------------------------------|
| `/`            | Vote            | Public       | Country cards grid; click to open vote modal                  |
| `/results`     | Live Results    | Public       | Horizontal bar chart, auto-refreshes every 10 s               |
| `/login`       | Login           | Public       | Token + role selector; redirects to `/admin` or `/jury`       |
| `/admin`       | Admin Dashboard | Admin token  | Toggle voting, reset votes, add country / artist / song       |
| `/jury`        | Jury Vote       | Jury token   | Eurovision-style point selector (1–8, 10, 12)                 |
| `/health`      | Health Check    | Public       | Returns `200 OK` — used by Docker Compose healthcheck         |
| `/api/results` | Results JSON    | Public       | JSON endpoint polled by the results page every 10 s           |
| `/*`           | Error Pages     | —            | 403, 404, 500 with [http.cat](https://http.cat) images        |

## Tech Stack

- **Python 3.12** + **Flask 3.x**
- **Jinja2** templates
- **Tailwind CSS** via CDN (`cdn.tailwindcss.com`)
- **Gunicorn** — 2 workers, 4 threads, `--preload` (production server)
- **requests** library for backend API communication
- **OpenTelemetry** — distributed tracing + custom metrics via `telemetry.py`

## Project Structure

```
frontend/
├── Dockerfile
├── README.md
└── src/
    ├── main.py              # Flask application (routes, API helpers)
    ├── telemetry.py         # OpenTelemetry tracing, metrics, and instrumentation setup
    ├── requirements.txt     # Python dependencies
    ├── static/
    │   ├── css/             # Custom stylesheets
    │   └── js/              # Custom scripts
    └── templates/
        ├── base.html        # Shared layout: navbar, flash messages, footer
        ├── vote.html        # Home / public vote page
        ├── results.html     # Live results with polling bar chart
        ├── login.html       # Login form (token + role)
        ├── admin.html       # Admin dashboard
        ├── jury.html        # Jury voting with point selectors
        └── error.html       # Error page (403, 404, 500)
```


## Observability

Telemetry is initialised in `telemetry.py` immediately after the Flask app object is created (before any routes are registered), ensuring all requests are instrumented:

- **Distributed Tracing** — `FlaskInstrumentor` and `RequestsInstrumentor` auto-instrument inbound requests and outbound backend calls. Traces are exported to the OTel Collector via OTLP/HTTP.
- **Custom Metrics** — the following application-level metrics are recorded:
  - `backend_call_duration_seconds` — duration of each backend API call, labelled by endpoint and status code
  - `votes_total` — counter of votes cast, labelled by type (`public` or `jury`)
  - `active_sessions` — gauge tracking the number of currently logged-in sessions
- **Prometheus** — metrics are exposed in multiprocess-safe mode (via `PROMETHEUS_MULTIPROC_DIR`) so all Gunicorn workers contribute to the same counters.

## Backend API Endpoints Used

| Method   | Endpoint               | Used By            | Purpose                       |
|----------|------------------------|--------------------|-------------------------------|
| `GET`    | `/songs/`              | Vote, Admin, Jury  | Fetch all songs with details  |
| `GET`    | `/votes/`              | Results            | Fetch vote totals             |
| `GET`    | `/countries/`          | Admin              | Fetch registered countries    |
| `POST`   | `/vote/`               | Vote               | Cast a public vote            |
| `GET`    | `/admin/authenticate`  | Login              | Validate an admin token       |
| `GET`    | `/jury/authenticate`   | Login              | Validate a jury token         |
| `POST`   | `/jury/vote/`          | Jury               | Cast a jury vote              |
| `POST`   | `/admin/open/`         | Admin              | Open voting                   |
| `POST`   | `/admin/close`         | Admin              | Close voting                  |
| `DELETE` | `/admin/deleteVotes/`  | Admin              | Reset all votes to zero       |
| `POST`   | `/admin/addCountry/`   | Admin              | Add a new country             |
| `POST`   | `/admin/addArtist/`    | Admin              | Add a new artist              |
| `POST`   | `/admin/addSong/`      | Admin              | Add a new song                |

## Shared Components

- **Navbar** — Logo left, nav links right, voting status pill (OPEN / CLOSED)
- **StatusBadge** — Animated yellow pill when voting is open; grey when closed
- **CountryCard** — Flag image (via flagcdn.com), country name, song title, points badge
- **Modal** — Vote form overlay with phone number input
- **Flash Messages** — Auto-dismissing toast notifications for success / error feedback
- **ProtectedRoute** — Server-side session check with role-based redirects (via `login_required` decorator)

## Authentication & Session

Roles are stored in the Flask server-side session after a successful login:

| Role    | Access                          |
|---------|---------------------------------|
| `admin` | Admin dashboard + Jury pages    |
| `jury`  | Jury voting page only           |

On login form submission, the token is verified against the backend **before** a session is created:

- `POST /login` with `role=admin` calls `GET /admin/authenticate?Token=<value>` on the CRUD API.
- `POST /login` with `role=jury` calls `GET /jury/authenticate?Token=<value>` on the CRUD API.

The backend returns `HTTP 202` on success or `HTTP 403` with an error message on failure. Only a `202` response causes the session to be established; any other response surfaces the backend's error message to the user via a flash notification and leaves the session untouched.

Tokens are passed through to the backend API via query parameters on subsequent protected requests (e.g. open/close voting, jury vote submission).
