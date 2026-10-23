# API image, and the one-shot goose image. Build from the repo root:
#   docker build -f deploy/docker/api.Dockerfile --target api .
#   docker build -f deploy/docker/api.Dockerfile --target migrate .

FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api \
	&& CGO_ENABLED=0 go build -trimpath -o /out/goose github.com/pressly/goose/v3/cmd/goose

# Debian so the image has a shell and curl. Compose probes /readyz with curl,
# and ca-certificates lets the API call R2 over HTTPS.
FROM debian:bookworm-slim AS api
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates curl \
	&& rm -rf /var/lib/apt/lists/* \
	&& useradd --system --uid 65532 --create-home --shell /usr/sbin/nologin api
COPY --from=build --chown=api:api /out/api /api
USER api
EXPOSE 8080
ENTRYPOINT ["/api"]

FROM debian:bookworm-slim AS migrate
RUN useradd --system --uid 65532 --create-home --shell /usr/sbin/nologin migrate
COPY --from=build /out/goose /usr/local/bin/goose
COPY db/migrations /migrations
COPY deploy/docker/migrate-entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh /usr/local/bin/goose
USER migrate
ENTRYPOINT ["/entrypoint.sh"]
CMD ["up"]
