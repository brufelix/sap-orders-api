FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz \
    | tar xz -C /usr/local/bin migrate

WORKDIR /app
COPY --from=builder /server /app/server
COPY migrations /app/migrations

EXPOSE 8081

CMD ["/app/server"]
