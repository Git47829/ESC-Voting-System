FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 1001 appgroup && \
    useradd --uid 1001 --gid appgroup --no-create-home appuser

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt && \
    pip install --no-cache-dir aio-pika redis[hiredis]

COPY src/main.py ./src/main.py
COPY src/rabbitmq_consumer.py ./src/rabbitmq_consumer.py
COPY src/telemetry.py ./src/telemetry.py

RUN chown -R appuser:appgroup /app

USER appuser

ENV PYTHONPATH=/app/src:$PYTHONPATH

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8880/health')"

CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8880"]
