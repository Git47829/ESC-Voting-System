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

## Access

The frontend is not exposed directly. All traffic is routed through the Caddy reverse proxy:

| URL | Description |
|-----|-------------|
| `https://<host>/` | Main voting UI |
| `https://<host>/now` | Running Now — live contest stage |
| `https://<host>/results` | Live results page |
| `https://<host>/login` | Login page |
| `https://<host>/admin` | Admin dashboard |
| `https://<host>/jury` | Jury voting page |

Caddy handles TLS termination using its internal CA. See the [Caddy README](../Caddy/README.md) for instructions on trusting the certificate.

## Pages

| Route                    | Page              | Auth        | Description                                                                         |
|--------------------------|-------------------|-------------|-------------------------------------------------------------------------------------|
| `/`                      | Vote              | Public      | Country cards grid; point stepper per song; bulk submit via modal                  |
| `/now`                   | Running Now       | Public      | Current contest song with YouTube embed, progress bar, and inline voting panel     |
| `/results`               | Live Results      | Public      | Horizontal bar chart, auto-refreshes every 10 s                                    |
| `/login`                 | Login             | Public      | Token + role selector; redirects to `/admin` or `/jury`                            |
| `/admin`                 | Admin Dashboard   | Admin token | Toggle voting, reset votes, start/advance contest, add country / artist / song     |
| `/jury`                  | Jury Vote         | Jury token  | Eurovision-style point selector (1–8, 10, 12)                                      |
| `/health`                | Health Check      | Public      | Returns `200 OK` — used by Docker Compose healthcheck                              |
| `/api/results`           | Results JSON      | Public      | JSON endpoint polled by the results page every 10 s                                |
| `/api/contest/current`   | Contest JSON      | Public      | JSON proxy for the current contest song; polled by `/now` every 5 s               |
| `/*`                     | Error Pages       | —           | 403, 404, 500 with [http.cat](https://http.cat) images                             |

> **Note:** `/api/results` and `/api/contest/current` are frontend-internal routes. Caddy's `/crud-api/*` proxy rule targets the CRUD DB API and uses a different prefix specifically to avoid colliding with this namespace.

## Running Now (`/now`)

The Running Now page is the live contest stage. It is designed to stay open on a shared screen while the admin advances through songs one by one.

### Features

- **YouTube embed** — full-width 16:9 iframe. If no URL is stored for a song, a placeholder is shown instead.
- **Progress bar** — shows how many songs have performed out of the total.
- **Song info card** — country flag emoji, song title, artist name, country name, and a live score counter.
- **Inline voting panel** — voters can allocate points without leaving the page:
  - Point stepper (+/− buttons, keyboard `+`/`-` shortcuts)
  - Phone number input (hashed server-side — used only for duplicate prevention)
  - Country selector (server-rendered from the songs list)
  - Submit button, loading spinner, and per-submission response message
- **Auto-advance** — the page polls `/api/contest/current` every 5 seconds. When the admin presses **Next Song**, all viewers detect the new song index and reload automatically.
- **Contest-end banner** — when all songs have performed, a banner links to the results page.
- **Admin controls** — when logged in as admin, a **Next Song** button appears in the page header.

### YouTube URL Handling

YouTube URLs are normalised to embed format (`https://www.youtube.com/embed/VIDEO_ID`) at two points:

1. **Admin form** (`/admin`) — the YouTube URL field runs `normalizeYoutubeInput()` on every keystroke. It extracts the video ID from any format (watch, share, short, embed) and replaces the field value with the canonical embed URL in real time. The hint text turns green on success or red if no ID can be found.
2. **Server side** (`main.py`) — `normalize_youtube_url()` re-normalises the stored URL at render time as a safety net, so even URLs that were inserted directly into the database without going through the form are displayed correctly.

Supported input formats:

| Format | Example |
|--------|---------|
| Watch URL | `https://www.youtube.com/watch?v=VIDEO_ID` |
| Short URL | `https://youtu.be/VIDEO_ID` |
| Embed URL | `https://www.youtube.com/embed/VIDEO_ID` |
| Shorts URL | `https://www.youtube.com/shorts/VIDEO_ID` |

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
    ├── main.py              # Flask application — routes, API helpers, URL normalizer
    ├── telemetry.py         # OpenTelemetry tracing, metrics, and instrumentation setup
    ├── requirements.txt     # Python dependencies
    ├── static/
    │   ├── css/             # Custom stylesheets
    │   └── js/              # Custom scripts
    └── templates/
        ├── base.html        # Shared layout: navbar, flash messages, footer
        ├── vote.html        # Home / public vote page (all songs, point stepper)
        ├── now.html         # Running Now — YouTube embed + inline voting
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

| Method   | Endpoint                    | Used By              | Purpose                                    |
|----------|-----------------------------|----------------------|--------------------------------------------|
| `GET`    | `/songs/`                   | Vote, Admin, Jury, Now | Fetch all songs with details             |
| `GET`    | `/votes/`                   | Results              | Fetch vote totals                          |
| `GET`    | `/countries/`               | Admin                | Fetch registered countries                 |
| `GET`    | `/contest/current/`         | Now                  | Fetch the current song in the active run   |
| `POST`   | `/vote/`                    | Vote, Now            | Cast a public vote                         |
| `GET`    | `/admin/authenticate`       | Login                | Validate an admin token                    |
| `GET`    | `/jury/authenticate`        | Login                | Validate a jury token                      |
| `POST`   | `/jury/vote/`               | Jury                 | Cast a jury vote                           |
| `POST`   | `/admin/open/`              | Admin                | Open voting                                |
| `POST`   | `/admin/close`              | Admin                | Close voting                               |
| `DELETE` | `/admin/deleteVotes/`       | Admin                | Reset all votes to zero                    |
| `POST`   | `/admin/addCountry/`        | Admin                | Add a new country                          |
| `POST`   | `/admin/addArtist/`         | Admin                | Add a new artist                           |
| `POST`   | `/admin/addSong/`           | Admin                | Add a new song (with optional YouTube URL) |
| `POST`   | `/admin/startContest/`      | Admin                | Shuffle all songs and start the contest    |
| `POST`   | `/admin/advanceContest/`    | Admin                | Advance to the next song                   |

All backend calls go directly to `db-crud-api:8000` over the internal Docker `frontend` network — they do not go through Caddy.

## Flask Routes

| Method | Route                       | Handler                  | Auth        | Description                                           |
|--------|-----------------------------|--------------------------|-------------|-------------------------------------------------------|
| `GET`  | `/`                         | `vote_page`              | Public      | Render the vote page with all songs                   |
| `GET`  | `/results`                  | `results_page`           | Public      | Render the results page                               |
| `GET`  | `/api/results`              | `api_results`            | Public      | JSON vote totals (polled by results page)             |
| `POST` | `/vote/submit`              | `submit_vote`            | Public      | Submit a public vote; forwards cookie to/from backend |
| `GET`  | `/now`                      | `now_playing`            | Public      | Render the Running Now page                           |
| `GET`  | `/api/contest/current`      | `api_contest_current`    | Public      | JSON proxy for the current contest song               |
| `GET`  | `/login`                    | `login`                  | Public      | Render the login form                                 |
| `POST` | `/login`                    | `login`                  | Public      | Process login; establish session on success           |
| `GET`  | `/logout`                   | `logout`                 | Public      | Clear session and redirect to `/`                     |
| `GET`  | `/admin`                    | `admin_dashboard`        | Admin       | Render the admin dashboard                            |
| `POST` | `/admin/open`               | `admin_open_vote`        | Admin       | Open voting                                           |
| `POST` | `/admin/close`              | `admin_close_vote`       | Admin       | Close voting                                          |
| `POST` | `/admin/reset`              | `admin_reset_votes`      | Admin       | Reset all votes                                       |
| `POST` | `/admin/add-country`        | `admin_add_country`      | Admin       | Add a country                                         |
| `POST` | `/admin/add-artist`         | `admin_add_artist`       | Admin       | Add an artist                                         |
| `POST` | `/admin/add-song`           | `admin_add_song`         | Admin       | Add a song (with optional YouTube URL)               |
| `POST` | `/admin/start-contest`      | `admin_start_contest`    | Admin       | Start the contest (shuffle + persist order)           |
| `POST` | `/admin/advance-contest`    | `admin_advance_contest`  | Admin       | Advance to the next song; redirect to `/now`          |
| `GET`  | `/jury`                     | `jury_page`              | Jury/Admin  | Render the jury voting page                           |
| `POST` | `/jury/submit`              | `jury_submit_vote`       | Jury/Admin  | Submit a jury vote                                    |

## Shared Components

- **Navbar** — Logo left, nav links right (Vote, Running Now with live dot, Results, Admin, Jury), voting status pill (OPEN / CLOSED with animated ping)
- **Running Now link** — Always visible in the navbar with a pulsing red dot to indicate a live stage
- **StatusBadge** — Animated yellow pill when voting is open; grey when closed
- **CountryCard** — Flag image (via flagcdn.com), country name, song title, points badge, point stepper
- **Modal** — Vote form overlay with phone number input and country selector (used on the Vote page)
- **Flash Messages** — Auto-dismissing toast notifications for success / error feedback (4 s fade animation)
- **ProtectedRoute** — Server-side session check with role-based redirects (via `login_required` decorator)
- **Confirmation Modals** — Used for destructive/irreversible actions: Reset All Votes, Start Contest

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

Tokens are passed through to the backend API via query parameters on subsequent protected requests (e.g. open/close voting, start/advance contest, jury vote submission).

## Container Security

The frontend container runs as a **non-root user** (`appuser`, UID 1001). The container has no exposed ports — all external traffic is routed through Caddy.