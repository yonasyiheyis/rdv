FROM alpine:3.20
LABEL org.opencontainers.image.source="https://github.com/yonasyiheyis/rdv"

# GoReleaser puts the compiled binary in the build context as "rdv"
COPY rdv /usr/local/bin/rdv
ENTRYPOINT ["/usr/local/bin/rdv"]
