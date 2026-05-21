FROM golang:1.22-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/input-cast ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /out/input-cast /app/input-cast
COPY configs ./configs
COPY web ./web

EXPOSE 8080

ENTRYPOINT ["/app/input-cast"]
