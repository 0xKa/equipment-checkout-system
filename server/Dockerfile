# syntax=docker/dockerfile:1

FROM golang:1.26.4-alpine3.23 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOBIN=/out go install \
    github.com/pressly/goose/v3/cmd/goose@v3.27.2

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -buildvcs=false \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/equipment-checkout \
    .

FROM alpine:3.23.5 AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app -h /app app

WORKDIR /app
USER app:app

FROM runtime AS migrate

COPY --from=build --chown=app:app /out/goose /usr/local/bin/goose
COPY --chown=app:app db/migrations /migrations

ENV GOOSE_DRIVER=postgres
ENV GOOSE_MIGRATION_DIR=/migrations

ENTRYPOINT ["/usr/local/bin/goose", "-no-color"]
CMD ["up"]

FROM runtime AS api

COPY --from=build --chown=app:app /out/equipment-checkout /usr/local/bin/equipment-checkout

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/equipment-checkout"]
CMD ["serve"]
