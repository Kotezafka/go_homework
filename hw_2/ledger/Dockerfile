FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.work ./
COPY ledger/go.mod ./ledger/
COPY auth/go.mod ./auth/
COPY gateway/go.mod ./gateway/

WORKDIR /app/ledger
RUN go mod download

WORKDIR /app
COPY ledger/ ./ledger/

WORKDIR /app/ledger
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ledger ./cmd/ledger

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/ledger/ledger .

EXPOSE 50052

CMD ["./ledger"]