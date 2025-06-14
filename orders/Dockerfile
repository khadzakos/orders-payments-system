FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main cmd/orders/main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client

COPY --from=builder /app/main .

COPY migrations /app/migrations

EXPOSE 8081

ENTRYPOINT ["./main"]