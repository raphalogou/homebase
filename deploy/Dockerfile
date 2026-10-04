# Build from the repository root:
#   docker build -f deploy/Dockerfile -t homebase .

FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS server
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web /src/web/dist ./internal/webui/dist
RUN apk add --no-cache ca-certificates \
 && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /homebase ./cmd/homebase \
 && mkdir /data

# busybox:musl: a 1.5 MB base that still has sh, stty (to hide the passphrase
# in hash-passphrase) and wget (for the health check). It has no CA
# certificates, which the push services need, so they come from the build
# stage. uid 65532 matches the README's backup folder instructions.
FROM busybox:musl
RUN echo 'homebase:x:65532:65532::/data:/bin/false' >> /etc/passwd \
 && echo 'homebase:x:65532:' >> /etc/group
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
