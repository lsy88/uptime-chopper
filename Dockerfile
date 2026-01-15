# Stage 1: Build the Frontend
FROM docker.1ms.run/node:18-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build the Backend
FROM docker.1ms.run/golang:1.24-alpine AS backend-builder
ENV GOPROXY=https://goproxy.cn,direct \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    TZ=Asia/Shanghai
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go build -ldflags '-s -w' -trimpath -o uptime-chopper ./cmd/server

# Stage 3: Final Image
FROM docker.1ms.run/alpine:latest

ENV UPTIME_CHOPPER_SERVE_FRONTEND=true \
    UPTIME_CHOPPER_HTTP_ADDR=:7601

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=backend-builder /app/uptime-chopper .
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN mkdir -p data

COPY config.yaml . 

EXPOSE 7601

CMD ["./uptime-chopper"]