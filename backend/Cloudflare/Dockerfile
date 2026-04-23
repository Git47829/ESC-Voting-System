FROM cloudflare/cloudflared:2024.4.1

COPY config.yml /etc/cloudflared/config.yml
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

