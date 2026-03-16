FROM golang:latest AS builder

WORKDIR /src

COPY src/ .

# go mod tidy generates go.sum and downloads dependencies in one step
RUN go mod tidy && go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /esc-points-converter .

# Inline healthcheck binary
RUN printf 'package main\n\nimport (\n\t"net/http"\n\t"os"\n)\n\nfunc main() {\n\tresp, err := http.Get("http://localhost:8090/health")\n\tif err != nil || resp.StatusCode != 200 {\n\t\tos.Exit(1)\n\t}\n\tos.Exit(0)\n}\n' > /tmp/healthcheck.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /healthcheck /tmp/healthcheck.go

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /esc-points-converter /esc-points-converter
COPY --from=builder /healthcheck /healthcheck

EXPOSE 8090

USER nonroot:nonroot

CMD ["/esc-points-converter"]
