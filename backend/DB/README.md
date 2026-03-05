# MySQL Database

Custom MySQL 8.0 image for the ESC Voting System. Runs the schema and seed data automatically on first start.

## Docker

The container exposes MySQL on port **3306** and is reachable inside the Docker network via the alias `mysql`.

## Environment Variables

| Variable              | Default        | Description                      |
|-----------------------|----------------|----------------------------------|
| `MYSQL_DATABASE`      | `esc_voting`   | Database name                    |
| `MYSQL_USER`          | `esc_user`     | Application user                 |
| `MYSQL_PASSWORD`      | `esc_password` | Application user password        |
| `MYSQL_ROOT_PASSWORD` | `secretroot`   | Root password                    |

> Values are set in `docker-compose.yaml`. Override them via a `.env` file in the project root.

## Schema

### `Land` — Countries

| Column | Type                  | Notes                     |
|--------|-----------------------|---------------------------|
| `ID`   | `CHAR(3)` PK          | ISO 3166-1 alpha-3 code   |
| `Name` | `VARCHAR(100)` UNIQUE | Country display name      |
| `POT`  | `TINYINT UNSIGNED`    | Pot/group assignment      |

### `Komponist` — Composers

| Column   | Type           | Notes         |
|----------|----------------|---------------|
| `ID`     | `INT` PK AI    | Auto-increment |
| `Vorname`| `VARCHAR(100)` | First name    |
| `Name`   | `VARCHAR(100)` | Last name     |

### `Kuenstler` — Artists

| Column    | Type                              | Notes                        |
|-----------|-----------------------------------|------------------------------|
| `ID`      | `INT` PK AI                       | Auto-increment               |
| `Vorname` | `VARCHAR(100)`                    | First name (optional)        |
| `Name`    | `VARCHAR(200)`                    | Last / group name            |
| `Typ`     | `ENUM('solo','duo','gruppe')`     | Performer type, default solo |
| `Land_ID` | `CHAR(3)` FK → `Land.ID`         | Country the artist represents|

### `Song` — Songs

| Column            | Type                    | Notes                                        |
|-------------------|-------------------------|----------------------------------------------|
| `ID`              | `INT` PK AI             | Auto-increment                               |
| `Name`            | `VARCHAR(200)`          | Song title                                   |
| `Land_ID`         | `CHAR(3)` FK → `Land`  | Competing country                            |
| `Kuenstler_ID`    | `INT` FK → `Kuenstler` | Performing artist                            |
| `PublikumsPunkte` | `SMALLINT UNSIGNED`     | Public vote points, default 0                |
| `JuryPunkte`      | `SMALLINT UNSIGNED`     | Jury vote points, default 0                  |
| `GesamtPunkte`    | `SMALLINT UNSIGNED`     | **Generated** = `PublikumsPunkte + JuryPunkte` (STORED) |

### `Song_Komponist` — Song ↔ Composer (many-to-many)

| Column        | Type                     | Notes           |
|---------------|--------------------------|-----------------|
| `Song_ID`     | `INT` FK → `Song`        | Composite PK    |
| `Komponist_ID`| `INT` FK → `Komponist`   | Composite PK    |

### `Voting_Status` — Global Voting State

| Column       | Type      | Notes                             |
|--------------|-----------|-----------------------------------|
| `VotingID`   | `INT` PK  | Always a single row (`ID = 1`)   |
| `isOpen`     | `BOOL`    | `TRUE` = voting open             |
| `lastChange` | `TIME`    | Time of last status change       |

### `Phone_Nums` — Hashed Phone Registry

| Column         | Type           | Notes                                  |
|----------------|----------------|----------------------------------------|
| `ID`           | `INT` PK AI    | Auto-increment                         |
| `Phone_Number` | `VARCHAR(200)` | bcrypt-hashed phone number (no duplicates enforced at DB level) |

## Seed Data

`seed_data.sql` is loaded automatically alongside the schema on first container start. It inserts:

- **Voting status**: open (`isOpen = TRUE`)
- **4 countries**: Germany, Sweden, France, Spain
- **3 composers**: Lena Meyer-Landrut, Thomas G:son, Aria Vidal
- **3 artists**: Lena Meyer-Landrut (solo/DEU), Alice Lindgren (duo/SWE), Jean Dupont (solo/FRA)
- **3 songs**: *Satellite Reprise*, *Northern Lights*, *Parisian Nights* — with pre-populated public and jury points

## Files

| File                   | Purpose                                              |
|------------------------|------------------------------------------------------|
| `Dockerfile`           | Custom MySQL 8.0 image                               |
| `db_scheme.sql`        | DDL — creates all tables                             |
| `seed_data.sql`        | DML — inserts initial countries, artists, and songs  |
| `docker-entrypoint.sh` | Entrypoint that applies schema + seed on first start |
| `healthcheck.sh`       | Container health check script used by Docker Compose |
| `my.cnf`               | Custom MySQL server configuration                    |

## Healthcheck

Docker Compose polls the container via `healthcheck.sh`. Dependent services (`api`, `otel-collector`) will not start until the database is healthy.
