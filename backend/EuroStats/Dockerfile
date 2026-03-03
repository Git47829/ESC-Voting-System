FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY proto ./proto

# Generate proto files
RUN python -m grpc_tools.protoc \
    -I./proto \
    --python_out=./src/proto \
    --grpc_python_out=./src/proto \
    ./proto/votes.proto && \
    touch ./src/proto/__init__.py

COPY src ./src

ENV PYTHONPATH=/app/src:$PYTHONPATH


HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:88000/health')"

CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "88000"]
