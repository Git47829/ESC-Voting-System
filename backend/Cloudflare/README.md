# Cloudflare Tunnel Setup Guide

This document provides step-by-step instructions for setting up Cloudflare Tunnel to expose the ESC Voting System to the internet.

## Prerequisites

- A domain managed by Cloudflare (or one you can transfer to Cloudflare)
- A Cloudflare account with access to Zero Trust features
- The ESC Voting System running locally via Docker Compose

## Step 1: Prepare Your Domain

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Add your domain (e.g., `escvoting.dev`) to Cloudflare if not already there
3. Ensure Cloudflare's nameservers are set as your domain's authoritative nameservers

## Step 2: Create the Tunnel

1. In Cloudflare Dashboard, navigate to **Zero Trust → Tunnels**
2. Click **Create a tunnel**
3. Choose **Cloudflared** as the connector type
4. Give your tunnel a name: `escvoting`
5. Click **Save** (you'll be prompted to install `cloudflared` on your machine, but we use Docker instead)
6. In the **Connectors** section, you'll see your tunnel token — **copy this entire token**

## Step 3: Configure `.env`

1. Open `.env` in the project root
2. Find the Cloudflare section (added by the setup script):
   ```env
   CLOUDFLARE_SECRET=
   ```
3. Paste your tunnel token into `CLOUDFLARE_SECRET`:
   ```env
   CLOUDFLARE_SECRET=eyJhIjoiZTYwNTYzMjM3OGRhYTc5Njc2YzY3OTY1Yzg0ZjU4ZjUiLCJ0IjoiZjUzOGY2ZTgtZGE0Ni00ZmI3LTk4ZDUtODM3ZmZkMzAwYTA5IiwicyI6IkpXRlNZWFJQZDJsSmMydHpPUSJ9
   ```
   (This is an example — use your actual token)

## Step 4: Configure Public Routes in Cloudflare Dashboard

Back in the Cloudflare Dashboard tunnel config:

1. Go to **Zero Trust → Tunnels → escvoting**
2. Click the **Public Hostname** tab
3. Add a public hostname:
   - **Domain**: `escvoting.dev`
   - **Service**: `HTTP` | `http://esc-frontend:3001`
   - Click **Save**

4. Add another public hostname for Grafana:
   - **Domain**: `escvoting.dev`
   - **Path**: `/grafana/*`
   - **Service**: `HTTPS` | `https://caddy:443`
   - Under **Additional application settings**, enable **Skip TLS Verify** (since Caddy uses self-signed certs internally)
   - Click **Save**

> **Alternative:** These routes are already configured in `backend/Cloudflare/config.yml`. The Docker `cloudflared` service uses this config file automatically. Only use the Cloudflare Dashboard if you prefer managing routes there instead.

## Step 5: Start the System

```bash
docker compose up -d --build
```

The `cloudflared` service will:
1. Start automatically
2. Read your tunnel credentials from the environment
3. Connect to the Cloudflare edge
4. Begin routing traffic from `escvoting.dev` to your local services

Monitor logs:
```bash
docker compose logs -f cloudflared
```

Expected output:
```
cloudflared | 2026-04-23T14:45:00Z INF Thank you for trying Cloudflare Tunnel. Improvements and feedback: https://community.cloudflare.com/c/cloudflare-tunnel
cloudflared | 2026-04-23T14:45:00Z INF Registered tunnel connection connectorID=<ID>
cloudflared | 2026-04-23T14:45:00Z INF Tunnel running. Success!
```

## Step 6: Test Access

1. Open `https://escvoting.dev/` in your browser
2. You should see the ESC Voting System frontend
3. Open `https://escvoting.dev/grafana/` to access Grafana (default login: `admin` / `admin`)

## Troubleshooting

### "Bad Request" or "502 Bad Gateway"

- Check that `cloudflared` is connected: `docker compose logs cloudflared`
- Verify the tunnel token in `.env` is correct
- Ensure Docker Compose services are healthy: `docker compose ps`

### "Could not resolve"

- Verify your domain's nameservers are pointing to Cloudflare
- Wait up to 48 hours for DNS propagation if you recently transferred the domain

### Self-signed Certificate Warning

Internally, Caddy uses a self-signed certificate to communicate between services. This is normal and secure because:
- All traffic is encrypted (HTTPS)
- Services only trust each other via Docker's internal network isolation
- Public users access via Cloudflare's TLS, not Caddy's internal cert

If you see SSL/TLS errors in `cloudflared` logs, add this to your tunnel config:
```yaml
originRequest:
  noTLSVerify: true
```
(This is already configured for Grafana in `backend/Cloudflare/config.yml`)

## Security Considerations

1. **Exposed Services**: Only frontend and Grafana are public. Database, APIs, and other services are not accessible via the tunnel.
2. **Credentials**: Keep `CLOUDFLARE_SECRET` secure. Do not commit it to git (`.gitignore` already excludes `.env`).
3. **Tunnel Token**: If leaked, rotate it immediately in the Cloudflare Dashboard.
4. **Grafana Access**: Consider adding Cloudflare's Zero Trust authentication to Grafana for additional security.

## Stopping the Tunnel

```bash
docker compose down
```

The tunnel will disconnect within seconds. Traffic to `escvoting.dev` will fail until services restart.

## Advanced: Firewall & Page Rules

Use Cloudflare's **Page Rules** or **WAF** to add additional security:

- Block access to `/grafana/*` from unknown IPs
- Rate limit `/voting` endpoints
- Add a custom error page if the tunnel is down

## References

- [Cloudflare Tunnel Documentation](https://developers.cloudflare.com/cloudflare-one/connections/connect-applications/)
- [Cloudflare Zero Trust](https://developers.cloudflare.com/cloudflare-one/)
- [Docker Cloudflare Image](https://hub.docker.com/r/cloudflare/cloudflared)
