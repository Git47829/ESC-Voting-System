FROM golang:latest

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY src/*.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /CRUD-DB-API

EXPOSE 8000

CMD ["./CRUD-DB-API"]
