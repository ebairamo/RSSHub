FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o rsshub ./cmd/rsshub

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/rsshub .
COPY --from=builder /app/migrations ./migrations

RUN apk --no-cache add ca-certificates

ENTRYPOINT ["./rsshub"]
CMD ["fetch"]