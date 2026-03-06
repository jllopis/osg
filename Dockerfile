# Multi-stage Dockerfile for OSG
# Stage 1: Build the Go binary
# Stage 2: Minimal runtime image

# --- Builder ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X osg/internal/app.Version=${VERSION} -X osg/internal/app.Commit=${COMMIT} -X osg/internal/app.Date=${DATE}" \
    -o /osg ./cmd/osg

# --- Runtime ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S osg && adduser -S osg -G osg

COPY --from=builder /osg /usr/local/bin/osg

# Data directory for SQLite databases and site content.
RUN mkdir -p /data && chown osg:osg /data
VOLUME ["/data"]

# Site content and config are expected to be mounted at /site.
RUN mkdir -p /site && chown osg:osg /site
WORKDIR /site

USER osg

# Default: run the API server. Override CMD to run build, serve, etc.
EXPOSE 8090
CMD ["osg", "api"]
