FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy source (including vendor directory)
COPY . .

# Cross-compile natively (no QEMU) — modernc.org/sqlite is pure Go
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -mod=vendor -o /simulator ./cmd/simulator

# Runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /simulator /app/simulator
COPY --from=builder /app/config /app/config

EXPOSE 8081
EXPOSE 4840-4880

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

ENTRYPOINT ["/app/simulator"]
