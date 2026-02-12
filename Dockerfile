# Build the manager binary
FROM golang:1.25.6-alpine3.23 AS builder

WORKDIR /app
# Install build tooling needed by Makefile
RUN apk add --no-cache make
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# copy the source code
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY Makefile Makefile
COPY config.toml config.toml

RUN make build

## ---------------------------------------------------
## Build the final image
FROM scratch

LABEL org.opencontainers.image.source="https://github.com/mfaizanse/ext-kyma-mcp"

WORKDIR /

COPY --from=builder /app/bin/ext-kyma-mcp .
COPY --from=builder /app/config.toml .

USER 65532:65532

EXPOSE 8080

# # Enable FIPS only mode and disable TLS ML-KEM as it is not FIPS compliant (https://pkg.go.dev/crypto/tls#Config.CurvePreferences)
# ENV GODEBUG=fips140=only,tlsmlkem=0

ENTRYPOINT ["/ext-kyma-mcp", "--config", "config.toml"]
