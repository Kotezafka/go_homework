FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

WORKDIR /app/cmd/ledger
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ledger main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/cmd/ledger/ledger .

CMD ["./ledger"]