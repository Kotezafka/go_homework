FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.work ./
COPY auth/go.mod ./auth/
COPY ledger/go.mod ./ledger/
COPY gateway/go.mod ./gateway/

WORKDIR /app/auth
RUN go mod download

WORKDIR /app
COPY auth/ ./auth/

WORKDIR /app/auth
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o auth ./cmd/auth/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/auth/auth .

EXPOSE 50051

CMD ["./auth"]