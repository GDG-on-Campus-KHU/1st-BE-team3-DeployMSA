FROM golang:1.23.1 AS builder
WORKDIR /app
# 상위 디렉토리의 모든 파일을 복사
COPY ../../ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o app ./cmd/client/main.go

FROM debian:bullseye-slim as deploy
RUN apt-get update && \
    apt-get install -y curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/app .
COPY --from=builder /app/.env .
# 실행 전에 URL 테스트
CMD sh -c 'echo "Testing video URL..." && \
    curl -I $VIDEO_URL && \
    echo "URL is accessible, starting app..." && \
    ./app'