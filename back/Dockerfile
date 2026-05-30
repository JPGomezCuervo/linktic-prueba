FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -o /out/app ./cmd/app \
 && go build -trimpath -o /out/migrate ./cmd/migrate

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache sqlite ca-certificates

COPY --from=build /out/app /app/app
COPY --from=build /out/migrate /app/migrate
COPY migrations /app/migrations
COPY db/seed.sql /app/seed.sql
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh && mkdir -p /app/db

EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
