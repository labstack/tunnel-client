FROM alpine:3.11

RUN apk add --no-cache ca-certificates

COPY dist/linux_amd64/tunnel /usr/local/bin/tunnel

ENTRYPOINT ["tunnel"]
