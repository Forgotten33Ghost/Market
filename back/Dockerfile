FROM golang:1.24-alpine AS builder

WORKDIR /api

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /api/server

FROM alpine:3.18

WORKDIR /api

COPY --from=builder /api/server /api/server

ENV DB_HOST=postgres
ENV DB_PORT=5432

EXPOSE 8080
CMD ["/api/server"]