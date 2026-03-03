FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY proto ./proto
COPY scripts/generate_proto.sh ./scripts/generate_proto.sh

RUN mkdir -p src/proto && \
    touch src/proto/__init__.py

RUN chmod +x ./scripts/generate_proto.sh && \
    ./scripts/generate_proto.sh

COPY src ./src

ENV PYTHONPATH=/app/src:$PYTHONPATH

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:88000/health')"

CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "88000"]
