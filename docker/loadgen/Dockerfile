FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o loadgen ./cmd/loadgen

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/loadgen .
COPY docker/loadgen/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 9092

ENTRYPOINT ["/entrypoint.sh"]
