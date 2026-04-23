#!/bin/sh
set -e

# Write the credentials file from environment variable
echo "$CLOUDFLARE_TUNNEL_TOKEN" > /etc/cloudflared/cred.json

# Run cloudflared tunnel
exec cloudflared tunnel --config /etc/cloudflared/config.yml run "$CLOUDFLARE_TUNNEL_NAME"
