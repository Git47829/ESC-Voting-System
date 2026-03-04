# ESC Voting System — Frontend

Flask + Jinja2 + Tailwind CSS (CDN) frontend for the Eurovision Song Contest Voting System.

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

| Route      | Page            | Auth         | Description                                          |
|------------|-----------------|--------------|------------------------------------------------------|
| `/`        | Vote            | Public       | Country cards grid; click to open vote modal          |
| `/results` | Live Results    | Public       | Horizontal bar chart, auto-refreshes every 10s        |
| `/login`   | Login           | Public       | Token + role selector; redirects to `/admin` or `/jury` |
| `/admin`   | Admin Dashboard | Admin token  | Toggle voting, reset votes, add country/artist/song   |
| `/jury`    | Jury Vote       | Jury token   | Eurovision-style point selector (1–8, 10, 12)         |
| `/*`       | Error Pages     | —            | 403, 404, 500 with [http.cat](https://http.cat) images |

## Tech Stack

- **Python 3.12** + **Flask 3.x**
- **Jinja2** templates
- **Tailwind CSS** via CDN (`cdn.tailwindcss.com`)
- **Gunicorn** for production serving
- **requests** library for backend API communication

## Project Structure

```
frontend/
├── Dockerfile
├── README.md
└── src/
    ├── main.py              # Flask application (routes, API helpers)
    ├── requirements.txt     # Python dependencies
    ├── static/
    │   ├── css/             # Custom stylesheets (if needed)
    │   └── js/              # Custom scripts (if needed)
    └── templates/
        ├── base.html        # Shared layout: navbar, flash messages, footer
        ├── vote.html        # Home / public vote page
        ├── results.html     # Live results with polling bar chart
        ├── login.html       # Login form (token + role)
        ├── admin.html       # Admin dashboard
        ├── jury.html        # Jury voting with point selectors
        └── error.html       # Error page (403, 404, 500)
```

## Running Locally

### With Docker Compose 

From the project root:

```sh
docker compose up --build
```

The frontend will be available at [http://localhost:5000](http://localhost:5000).


The app will start on [http://localhost:5000](http://localhost:5000).

## Environment Variables

| Variable           | Default                    | Description                              |
|--------------------|----------------------------|------------------------------------------|
| `FLASK_SECRET_KEY` | `esc-voting-secret-key-change-me` | Secret key for session signing    |
| `API_BASE_URL`     | `http://db-crud-api:8000`  | Backend CRUD API base URL                |
| `API_TIMEOUT`      | `10`                       | Timeout in seconds for backend API calls |
| `FLASK_PORT`       | `5000`                     | Port the Flask app listens on            |
| `FLASK_DEBUG`      | `false`                    | Enable debug mode (`true` / `false`)     |

## Backend API Endpoints Used

| Method   | Endpoint               | Used By     | Purpose                    |
|----------|------------------------|-------------|----------------------------|
| `GET`    | `/songs/`              | Vote, Admin, Jury | Fetch all songs with details |
| `GET`    | `/votes/`              | Results     | Fetch vote totals          |
| `GET`    | `/countries/`          | Admin       | Fetch registered countries |
| `POST`   | `/vote/`               | Vote        | Cast a public vote         |
| `POST`   | `/jury/vote/`          | Jury        | Cast a jury vote           |
| `POST`   | `/admin/open/`         | Admin       | Open voting                |
| `POST`   | `/admin/close`         | Admin       | Close voting               |
| `DELETE` | `/admin/deleteVotes/`  | Admin       | Reset all votes to zero    |
| `POST`   | `/admin/addCountry/`   | Admin       | Add a new country          |
| `POST`   | `/admin/addArtist/`    | Admin       | Add a new artist           |
| `POST`   | `/admin/addSong/`      | Admin       | Add a new song             |

## Shared Components

- **Navbar** — Logo left, nav links right, voting status pill (OPEN/CLOSED)
- **StatusBadge** — Animated yellow pill when voting is open; grey when closed
- **CountryCard** — Flag image (via flagcdn.com), country name, song title, points badge
- **Modal** — Vote form overlay with phone number input
- **Flash Messages** — Auto-dismissing toast notifications for success/error feedback
- **ProtectedRoute** — Server-side session check with role-based redirects
