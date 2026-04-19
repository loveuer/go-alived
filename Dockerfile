# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w -X github.com/loveuer/go-alived/internal/cmd.Version=${VERSION}" \
    -o /out/go-alived .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    mkdir -p /etc/go-alived /etc/go-alived/scripts

COPY --from=builder /out/go-alived /usr/local/bin/go-alived

ENTRYPOINT ["/usr/local/bin/go-alived"]
CMD ["run", "--config", "/etc/go-alived/config.yaml"]
