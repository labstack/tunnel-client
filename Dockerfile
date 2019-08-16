FROM alpine:3.10

RUN apk add --no-cache ca-certificates

COPY dist/tunnel_linux_amd64/tunnel /usr/local/bin/tunnel

ENTRYPOINT ["tunnel"]
