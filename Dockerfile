FROM golang:1.24.1-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata wget

RUN wget -O /tini https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-amd64 && \
    chmod +x /tini

RUN adduser -D -g '' go

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o executor ./command/server/main.go

FROM scratch

ARG APP_PORT=3000

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /usr/bin/wget /usr/bin/wget

COPY --from=builder /tini /tini

COPY --from=builder /build/executor /executor

USER go

EXPOSE ${APP_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${APP_PORT}/health || exit 1

ENTRYPOINT ["/tini", "--",]

CMD ["/executor"]
