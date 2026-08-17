FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 gofer

COPY --from=build /out/server /usr/local/bin/gofer-server

USER gofer
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gofer-server", "-config", "/etc/gofer/config.yaml"]
