# NovaMail 多阶段构建
# 构建前端 → 嵌入 Go 二进制 → 最小运行镜像

# ---------- 阶段 1：前端构建 ----------
FROM node:22-alpine AS frontend
WORKDIR /build

COPY web/package.json web/package-lock.json* web/pnpm-lock.yaml* ./
RUN if [ -f pnpm-lock.yaml ]; then \
        corepack enable && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then \
        npm ci; \
    else \
        npm install; \
    fi

COPY web/ ./
RUN if command -v pnpm >/dev/null 2>&1; then pnpm build; else npm run build; fi

# ---------- 阶段 2：后端构建 ----------
FROM golang:1.25-alpine AS backend
WORKDIR /build

# 嵌入前端产物（vite outDir=../server/dist，在前端阶段实际落盘 /server/dist）
COPY --from=frontend /server/dist/ ./embedfs/dist/

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath \
    -ldflags "-s -w -X novamail/buildinfo.Version=${APP_VERSION} -X main.isProduction=true" \
    -o /novamail .

# ---------- 阶段 3：运行镜像 ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S novamail && adduser -S novamail -G novamail

WORKDIR /app
COPY --from=backend /novamail /app/novamail

RUN mkdir -p /app/data && chown -R novamail:novamail /app

USER novamail

ENV NOVAMAIL_HOST=0.0.0.0 \
    NOVAMAIL_PORT=8080 \
    NOVAMAIL_DSN=/app/data/novamail.db \
    TZ=Asia/Shanghai

VOLUME ["/app/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${NOVAMAIL_PORT}/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/novamail"]
