FROM rust:slim-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY Cargo.toml Cargo.lock ./

RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release && rm -rf src

COPY src/ ./src/

RUN touch src/main.rs && cargo build --release

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r appgroup && useradd -r -g appgroup -s /sbin/nologin appuser

COPY --from=builder /src/target/release/EuroMail /euromail

EXPOSE 3000

USER appuser:appgroup

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["curl", "-f", "http://localhost:3000/health"]

CMD ["/euromail"]
