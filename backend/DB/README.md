# MySQL Database (`backend/DB`)

Custom MySQL image (base: `mysql:8.2`) for ESC Voting.

## Startup

```bash
docker compose up -d db
```

The service is reachable in Docker as `mysql:3306`.

## Environment variables

| Variable | Default |
|---|---|
| `MYSQL_DATABASE` | `esc_voting` |
| `MYSQL_USER` | `esc_user` |
| `MYSQL_PASSWORD` | `esc_password` |
| `MYSQL_ROOT_PASSWORD` | `secretroot` |

## Initialization behavior

- `db_scheme.sql` and `seed_data.sql` are copied to `/docker-entrypoint-initdb.d/`
- MySQL applies them **only on first startup of a fresh data volume**
- Entrypoint runs MySQL with `--bind-address=0.0.0.0 --skip-name-resolve`

## Schema (tables)

- `Land`
- `Komponist`
- `Kuenstler`
- `Song` (includes generated `GesamtPunkte`, optional `YoutubeURL`)
- `Song_Komponist`
- `Voting_Status`
- `Contest_Run`
- `Phone_Nums` (`votes_remaining`, `votes_cast`)

## Seed data

`seed_data.sql` inserts:

- `Voting_Status` row (`VotingID=1`, `isOpen=TRUE`)
- countries: `DE`, `SE`, `FR`, `ES`
- composers/artists
- three songs with initial public/jury points and two YouTube embed URLs

## Healthcheck

`healthcheck.sh` uses:

```bash
mysqladmin ping -h 127.0.0.1 --protocol=TCP -u root -psecretroot
```

## Resetting schema/seed

If schema files changed and data volume already exists, recreate the DB volume, then start `db` again.
