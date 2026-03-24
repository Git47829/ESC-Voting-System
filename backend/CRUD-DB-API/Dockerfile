FROM golang:latest AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY main.go ./
COPY server/ ./server/
COPY proto/ ./proto/

RUN CGO_ENABLED=0 GOOS=linux go build -o /crud-db-api .

RUN printf 'package main\n\nimport (\n\t"net/http"\n\t"os"\n)\n\nfunc main() {\n\tresp, err := http.Get("http://localhost:8000/health")\n\tif err != nil || resp.StatusCode != 200 {\n\t\tos.Exit(1)\n\t}\n\tos.Exit(0)\n}\n' > /tmp/healthcheck.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /healthcheck /tmp/healthcheck.go

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /crud-db-api /crud-db-api
COPY --from=builder /healthcheck /healthcheck

EXPOSE 8000

USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD ["/healthcheck"]

CMD ["/crud-db-api"]
