# syntax=docker/dockerfile:1.7

# =============================================================================
# Sub2API Multi-Stage Dockerfile
# =============================================================================
# Stage 1: Build frontend
# Stage 2: Build integrated sub2apipay
# Stage 3: Build runtime Prisma CLI
# Stage 4: Build Go backend with embedded frontend
# Stage 5: PostgreSQL client
# Stage 6: Final runtime image
# =============================================================================

ARG NODE_IMAGE=node:24-alpine
ARG GOLANG_IMAGE=golang:1.26.2-alpine
ARG ALPINE_IMAGE=alpine:3.21
ARG POSTGRES_IMAGE=postgres:18-alpine
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ARG PNPM_VERSION=10.30.3
ARG APP_UID=1000
ARG APP_GID=1000

# -----------------------------------------------------------------------------
# Stage 1: Frontend Builder
# -----------------------------------------------------------------------------
FROM ${NODE_IMAGE} AS frontend-builder

WORKDIR /app/frontend
ENV PNPM_HOME=/pnpm
ENV PATH=${PNPM_HOME}:${PATH}

# Install pnpm
RUN corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate

# Install dependencies first (better caching; .npmrc affects resolution / script policy)
COPY --link frontend/package.json frontend/pnpm-lock.yaml frontend/.npmrc ./
RUN --mount=type=cache,id=sub2api-frontend-pnpm-store,target=/pnpm/store \
    pnpm config set store-dir /pnpm/store && \
    pnpm install --frozen-lockfile

# Copy frontend source and build
COPY --link frontend/ ./
RUN pnpm run build

# -----------------------------------------------------------------------------
# Stage 2: Integrated Payment Builder
# -----------------------------------------------------------------------------
FROM ${NODE_IMAGE} AS pay-builder

WORKDIR /app/sub2apipay

ENV NEXT_TELEMETRY_DISABLED=1
ENV PNPM_HOME=/pnpm
ENV PATH=${PNPM_HOME}:${PATH}

RUN apk add --no-cache libc6-compat
RUN corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate

COPY --link sub2apipay/package.json sub2apipay/pnpm-lock.yaml sub2apipay/pnpm-workspace.yaml ./
RUN --mount=type=cache,id=sub2api-pay-pnpm-store,target=/pnpm/store \
    pnpm config set store-dir /pnpm/store && \
    pnpm install --frozen-lockfile

COPY --link sub2apipay/ ./
RUN pnpm exec prisma generate
RUN pnpm run build

# -----------------------------------------------------------------------------
# Stage 3: Prisma Runtime Builder
# -----------------------------------------------------------------------------
FROM ${NODE_IMAGE} AS prisma-runtime-builder

WORKDIR /app/prisma-runtime

COPY --link sub2apipay/package.json /tmp/sub2apipay-package.json
RUN PRISMA_VERSION="$(node -p "const pkg=require('/tmp/sub2apipay-package.json'); const version=pkg.dependencies?.prisma ?? pkg.devDependencies?.prisma; if (!version) throw new Error('prisma version not found in package.json'); if (!/^\\d+\\.\\d+\\.\\d+(?:[-+].+)?$/.test(version)) throw new Error('prisma version must be pinned exactly in package.json'); version")" && \
    node -e "const fs=require('node:fs'); const version=process.argv[1]; fs.writeFileSync('package.json', JSON.stringify({ name: 'sub2apipay-prisma-runtime', private: true, dependencies: { prisma: version } }, null, 2) + '\n')" "${PRISMA_VERSION}"
RUN --mount=type=cache,id=sub2api-prisma-npm-cache,target=/root/.npm \
    npm install --omit=dev --omit=optional --prefer-offline --no-audit --no-fund

# -----------------------------------------------------------------------------
# Stage 4: Backend Builder
# -----------------------------------------------------------------------------
FROM ${GOLANG_IMAGE} AS backend-builder

# Build arguments for version info (set by CI)
ARG VERSION=
ARG COMMIT=docker
ARG DATE
ARG GOPROXY
ARG GOSUMDB

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app/backend

# Copy go mod files first (better caching)
COPY --link backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy backend source first
COPY --link backend/ ./

# Copy frontend dist from previous stage (must be after backend copy to avoid being overwritten)
COPY --link --from=frontend-builder /app/backend/internal/web/dist ./internal/web/dist

# Build the binary (BuildType=release for CI builds, embed frontend)
# Version precedence: build arg VERSION > cmd/server/VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    VERSION_VALUE="${VERSION}" && \
    if [ -z "${VERSION_VALUE}" ]; then VERSION_VALUE="$(tr -d '\r\n' < ./cmd/server/VERSION)"; fi && \
    DATE_VALUE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}" && \
    CGO_ENABLED=0 GOOS=linux go build \
    -tags embed \
    -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.Commit=${COMMIT} -X main.Date=${DATE_VALUE} -X main.BuildType=release" \
    -o /app/sub2api \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 5: PostgreSQL Client (version-matched with docker-compose)
# -----------------------------------------------------------------------------
FROM ${POSTGRES_IMAGE} AS pg-client

# -----------------------------------------------------------------------------
# Stage 6: Final Runtime Image
# -----------------------------------------------------------------------------
FROM ${ALPINE_IMAGE}

ARG APP_UID
ARG APP_GID

# Labels
LABEL maintainer="Wei-Shaw <github.com/Wei-Shaw>"
LABEL description="Sub2API - AI API Gateway Platform"
LABEL org.opencontainers.image.source="https://github.com/Wei-Shaw/sub2api"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    su-exec \
    nodejs \
    wget \
    libpq \
    zstd-libs \
    lz4-libs \
    krb5-libs \
    libldap \
    libedit \
    && rm -rf /var/cache/apk/*

# Copy pg_dump and psql from the same postgres image used in docker-compose
# This ensures version consistency between backup tools and the database server
COPY --link --from=pg-client /usr/local/bin/pg_dump /usr/local/bin/pg_dump
COPY --link --from=pg-client /usr/local/bin/psql /usr/local/bin/psql
COPY --link --from=pg-client /usr/local/lib/libpq.so.5* /usr/local/lib/

# Create non-root user
RUN addgroup -g ${APP_GID} sub2api && \
    adduser -u ${APP_UID} -G sub2api -s /bin/sh -D sub2api

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder --chown=${APP_UID}:${APP_GID} /app/sub2api /app/sub2api
COPY --from=pay-builder --chown=${APP_UID}:${APP_GID} /app/sub2apipay/.next/standalone /app/sub2apipay
COPY --from=pay-builder --chown=${APP_UID}:${APP_GID} /app/sub2apipay/.next/static /app/sub2apipay/.next/static
COPY --from=pay-builder --chown=${APP_UID}:${APP_GID} /app/sub2apipay/public /app/sub2apipay/public
COPY --from=pay-builder --chown=${APP_UID}:${APP_GID} /app/sub2apipay/prisma /app/sub2apipay/prisma
COPY --from=pay-builder --chown=${APP_UID}:${APP_GID} /app/sub2apipay/prisma.config.ts /app/sub2apipay/prisma.config.ts
COPY --from=prisma-runtime-builder --chown=${APP_UID}:${APP_GID} /app/prisma-runtime/node_modules /app/node_modules

# Create data directory
RUN mkdir -p /app/data /app/sub2apipay

# Copy entrypoint script (fixes volume permissions then drops to sub2api)
COPY --link deploy/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Expose port (can be overridden by SERVER_PORT env var)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q -T 5 -O /dev/null http://localhost:${SERVER_PORT:-8080}/health || exit 1

# Run the application (entrypoint fixes /app/data ownership then execs as sub2api)
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
