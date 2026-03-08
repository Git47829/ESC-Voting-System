FROM golang:latest AS builder

WORKDIR /src

COPY /src/go.mod /src/go.sum ./

RUN go mod download

COPY src/*.go ./
COPY src/proto ./proto

RUN CGO_ENABLED=0 GOOS=linux go build -o /crud-db-api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /crud-db-api /crud-db-api

EXPOSE 8000

USER nonroot:nonroot

CMD ["/crud-db-api"]
