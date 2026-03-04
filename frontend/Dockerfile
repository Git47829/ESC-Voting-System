FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl && \
    rm -rf /var/lib/apt/lists/*

COPY src/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Directory for prometheus_client multiprocess mode.
# Each gunicorn worker writes its metrics here; the /metrics endpoint
# aggregates across all workers before returning the response.
RUN mkdir -p /tmp/prometheus_multiproc

COPY src/ .

ENV FLASK_SECRET_KEY=change-me-in-production
ENV API_BASE_URL=http://db-crud-api:8000
ENV API_TIMEOUT=10
ENV FLASK_PORT=5000
# Tell prometheus_client to use multiprocess mode
ENV PROMETHEUS_MULTIPROC_DIR=/tmp/prometheus_multiproc

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=10s \
    CMD curl -f http://localhost:5000/health || exit 1

# 2 workers is enough for MVP and keeps prometheus multiproc overhead low.
# --preload loads the app once before forking so OTel SDK is initialised
# in the master process, not re-initialised 2x in each worker.
CMD ["gunicorn", \
     "--bind", "0.0.0.0:5000", \
     "--workers", "2", \
     "--threads", "4", \
     "--timeout", "120", \
     "--preload", \
     "--access-logfile", "-", \
     "--error-logfile", "-", \
     "main:app"]
