FROM golang:1.23.1 AS builder
WORKDIR /app
COPY . .
RUN go mod download && \
go build -trimpath -ldflags "-w -s" -o app ./cmd/client

FROM debian:bullseye-slim as deploy
RUN apt-get update
COPY --from=builder /app/app .
CMD ["./app"]
