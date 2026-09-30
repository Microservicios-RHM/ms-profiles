FROM golang:1.26-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/perfiles-service ./cmd/server

FROM alpine:3.21
WORKDIR /app

RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/perfiles-service ./perfiles-service

USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1

CMD ["./perfiles-service"]
