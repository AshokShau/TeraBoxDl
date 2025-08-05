# Build stage
FROM golang:1.24.0-alpine3.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o teraboxdl .

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/teraboxdl .

CMD ["./teraboxdl"]
