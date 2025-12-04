FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.work ./
COPY ledger/go.mod ledger/go.sum ./ledger/
COPY gateway/go.mod ./gateway/

WORKDIR /app/ledger
RUN go mod download

WORKDIR /app/gateway
RUN go mod download

WORKDIR /app
COPY ledger/ ./ledger/
COPY gateway/ ./gateway/

WORKDIR /app/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gateway main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /app/gateway/gateway .

EXPOSE 8080

CMD ["./gateway"]
