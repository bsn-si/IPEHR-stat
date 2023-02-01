FROM golang:1.19.0-alpine3.16 AS build

RUN apk update && \
    apk add --no-cache gcc musl-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /srv

COPY ./ .

RUN go mod download

RUN go build -o ./bin/ipehr-stats ./cmd/main.go

FROM alpine:3.16

WORKDIR /srv

COPY --from=build /srv/bin/ /srv
COPY --from=build /srv/config.json.example /srv/config.json
COPY --from=build /srv/db /srv/db
COPY --from=build /srv/pkg/contracts /srv/contracts

CMD ["./ipehr-stats", "-config=./config.json"]
