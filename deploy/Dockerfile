# Build from the repository root:
#   docker build -f deploy/Dockerfile -t homebase \
#     --build-arg VERSION=$(git describe --tags --always) --build-arg COMMIT=$(git rev-parse --short HEAD) .
# For amd64 and arm64 at once (CI does this):
#   docker buildx build --platform linux/amd64,linux/arm64 -f deploy/Dockerfile .
#
# The build stages run on the build machine's platform and Go cross-compiles,
# and the last stage runs nothing, so no emulation is needed.

FROM --platform=$BUILDPLATFORM node:24-alpine AS web
ARG VERSION=dev COMMIT=unknown
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN VITE_VERSION=$VERSION VITE_COMMIT=$COMMIT npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS server
ARG TARGETOS TARGETARCH VERSION=dev COMMIT=unknown
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web /src/web/dist ./internal/webui/dist
RUN apk add --no-cache ca-certificates \
 && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags="-s -w -X main.version=$VERSION -X main.commit=$COMMIT" -o /homebase ./cmd/homebase \
 && mkdir /data \
 && printf 'root:x:0:0:root:/root:/bin/sh\nhomebase:x:65532:65532::/data:/bin/false\n' > /passwd \
 && printf 'root:x:0:\nhomebase:x:65532:\n' > /group

# busybox:musl: a 1.5 MB base that still has sh, stty (to hide the passphrase
# in hash-passphrase) and wget (for the health check). It has no CA
# certificates, which the push services need, so they come from the build
# stage, as do the user files (a RUN here would need emulation for arm64).
# uid 65532 matches the README's backup folder instructions.
FROM busybox:musl
COPY --from=server /passwd /etc/passwd
COPY --from=server /group /etc/group
COPY --from=server /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=server /homebase /usr/local/bin/homebase
COPY --from=server --chown=65532:65532 /data /data
ENV HOMEBASE_DATA=/data \
    HOMEBASE_ADDR=:8080
VOLUME /data
EXPOSE 8080
USER homebase
HEALTHCHECK --interval=1m --timeout=5s CMD wget -qO /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["homebase"]
CMD ["serve"]
