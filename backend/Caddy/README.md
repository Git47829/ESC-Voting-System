# Caddy Reverse Proxy

Caddy acts as the **sole external entry point** for the ESC Voting System. It terminates HTTPS, issues certificates via its built-in local CA, and routes incoming requests to the correct internal service by subpath.

No other service in the stack has ports bound to the host — all external traffic must pass through Caddy.

## Configuration

The Caddyfile is located at `src/Caddyfile` and is mounted into the container as read-only.

Two named Docker volumes persist Caddy's state across restarts:

| Volume | Purpose |
|---|---|
| `caddy_data` | TLS certificate storage, CA keys, OCSP cache |
| `caddy_config` | Autosaved runtime configuration |

## Ports

| Port | Protocol | Purpose |
|---|---|---|
| `80` | HTTP | Redirects all traffic to HTTPS (`308 Permanent`) |
| `443` | HTTPS | Main ingress — TLS termination + reverse proxy |

Both ports are bound on all interfaces (`0.0.0.0`) so the stack is reachable from other machines on the LAN.

## TLS

Caddy uses `tls internal` with `on_demand` to issue certificates from its own local CA (`pki.ca.local`). On first connection, Caddy obtains a certificate for the identifier presented in the TLS handshake (hostname or IP) and caches it for subsequent requests. The CA root certificate is stored in the `caddy_data` volume at:

```
/data/caddy/pki/authorities/local/root.crt
```

Because this is a private CA, browsers will show a security warning until the root certificate is trusted on the client machine.

### Trusting the CA Certificate

**Step 1 — copy the cert out of the container** (run on the server):

```bash
docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-local-ca.crt
```

**Step 2 — transfer it to your local machine** (if connecting remotely):

```bash
scp user@<server-ip>:~/Projects/ESC-Voting-System/caddy-local-ca.crt ./caddy-local-ca.crt
```

**Step 3 — install it** according to your OS:

**macOS**
```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ./caddy-local-ca.crt
```
Then fully quit and reopen your browser.

**Windows**
Double-click `caddy-local-ca.crt` → *Install Certificate* → *Local Machine* → *Place all certificates in the following store* → *Trusted Root Certification Authorities* → Finish. Restart your browser.

**Linux (Debian / Ubuntu)**
```bash
sudo cp caddy-local-ca.crt /usr/local/share/ca-certificates/caddy-local-ca.crt
sudo update-ca-certificates
```
Then restart your browser.

**Firefox (all platforms)**
Firefox maintains its own trust store independently of the OS. Go to:
*Settings → Privacy & Security → Certificates → View Certificates → Authorities → Import*
Select `caddy-local-ca.crt` and tick *Trust this CA to identify websites*.

## Routing

All routes live inside a single `https://` site block. Requests are matched top-to-bottom; the first matching `handle` or `handle_path` wins.

| Incoming path | Handler | Strip prefix? | Upstream |
|---|---|---|---|
| `/crud-api/*` | `handle_path` | yes — `/crud-api` stripped | `db-crud-api:8000` |
| `/eurostats/*` | `handle_path` | yes — `/eurostats` stripped | `eurostats:8880` |
| `/grafana*` | `handle` | no | `grafana:3000` |
| `/prometheus*` | `handle_path` | yes — `/prometheus` stripped | `prometheus:9090` |
| `/tempo/*` | `handle_path` | yes — `/tempo` stripped | `tempo:3200` |
| `/loki/*` | `handle_path` | yes — `/loki` stripped | `loki:3100` |
| `*` (catch-all) | `handle` | no | `esc-frontend:5000` |

All frontend routes (`/`, `/now`, `/results`, `/login`, `/admin`, `/jury`, `/vote/submit`, `/api/*`) are served by the catch-all rule and forwarded to the Flask frontend at `esc-frontend:5000`.

### Why some routes strip and others do not

- **`handle_path` (strip)** — used for services that serve everything from `/` internally and have no knowledge of any prefix. Caddy removes the prefix before forwarding so the upstream sees the original path.
- **`handle` (no strip)** — used for services that are configured to serve under a specific prefix themselves. Grafana is started with `GF_SERVER_ROOT_URL=.../grafana` and `GF_SERVER_SERVE_FROM_SUB_PATH=true`, so it already emits all asset URLs and redirects with `/grafana` baked in. Stripping the prefix would break navigation.

### OTel ingestion ports

The OTel Collector's OTLP ingestion ports (`4317` gRPC, `4318` HTTP) are **not routed through Caddy** and are not exposed to the host at all. Application services push telemetry directly to `otel-collector:4317` / `otel-collector:4318` over the internal `observability` Docker network.

## Docker Networks

Caddy is attached to two networks so it can reach all upstream services:

| Network | Upstreams reachable |
|---|---|
| `frontend` | `esc-frontend:5000`, `db-crud-api:8000` |
| `observability` | `eurostats:8880`, `grafana:3000`, `prometheus:9090`, `tempo:3200`, `loki:3100` |

## Reloading Configuration

Caddy watches its config file and can reload without downtime:

```bash
docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile
```

Or do a full container restart (brief interruption):

```bash
docker compose restart caddy
```

## Clearing TLS State

If you need to force Caddy to re-issue all certificates (e.g. after changing the Caddyfile site address), remove the data and config volumes:

```bash
docker compose down caddy
docker volume rm $(docker volume ls -q | grep caddy)
docker compose up -d caddy
```
