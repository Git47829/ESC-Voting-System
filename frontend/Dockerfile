FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl && \
    rm -rf /var/lib/apt/lists/*

COPY src/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt gunicorn==23.0.0

COPY src/ .

ENV FLASK_SECRET_KEY=change-me-in-production
ENV API_BASE_URL=http://db-crud-api:8000
ENV API_TIMEOUT=10
ENV FLASK_PORT=5000

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=5s \
    CMD curl -f http://localhost:5000/ || exit 1

CMD ["gunicorn", \
     "--bind", "0.0.0.0:5000", \
     "--workers", "4", \
     "--threads", "2", \
     "--timeout", "120", \
     "--access-logfile", "-", \
     "--error-logfile", "-", \
     "main:app"]
