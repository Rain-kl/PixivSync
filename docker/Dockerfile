# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25
ARG NODE_VERSION=22
ARG ALPINE_VERSION=3.23

FROM node:${NODE_VERSION}-alpine AS frontend-builder

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
ENV NODE_ENV=production
ENV NEXT_STANDALONE_EXPORT=true

RUN corepack enable && corepack prepare pnpm@10.10.0 --activate

WORKDIR /workspace/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm build:embed

FROM golang:${GO_VERSION}-alpine AS backend-builder

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /workspace/frontend/out ./internal/router/dist

RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags embed_frontend \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/wavelet \
    ./main.go

FROM alpine:${ALPINE_VERSION}

ARG TZ=Asia/Shanghai

RUN apk add --no-cache ca-certificates tzdata postgresql-client && \
    cp /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo "${TZ}" > /etc/timezone

WORKDIR /app

COPY --from=backend-builder /out/wavelet ./wavelet
COPY docs ./docs

EXPOSE 8000

ENTRYPOINT ["./wavelet"]
CMD ["all"]
