FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/rdv ./cmd/rdv

FROM alpine:3.20
LABEL org.opencontainers.image.source="https://github.com/yonasyiheyis/rdv"
COPY --from=builder /out/rdv /usr/local/bin/rdv
ENTRYPOINT ["/usr/local/bin/rdv"]