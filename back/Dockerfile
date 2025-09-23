FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /api

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api/server

FROM alpine:latest
WORKDIR /app

COPY --from=builder /api/server ./server

EXPOSE 8080

CMD ["./server"]
