FROM golang:latest

WORKDIR /src

COPY /src/go.mod /src/go.sum ./

RUN go mod download

COPY src/*.go ./
COPY src/proto ./proto

RUN CGO_ENABLED=0 GOOS=linux go build -o /crud-db-api

EXPOSE 8000

CMD ["/crud-db-api"]
