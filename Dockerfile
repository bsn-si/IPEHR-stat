FROM golang:1.19.0-alpine3.16 AS build

RUN apk update && \
    apk add --no-cache gcc musl-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /srv

COPY ./ .
COPY config.json.example config.json

RUN go mod download

RUN go build -o ./bin/ipehr-stats ./cmd/main.go

FROM alpine:3.16

WORKDIR /srv

COPY --from=build /srv/bin/ /srv
COPY --from=build /srv/config.json /srv
COPY --from=build /srv/pkg/contracts /srv/contracts
COPY --from=build /srv/db /srv/db

CMD ["./ipehr-stats", "-config=./config.json"]
