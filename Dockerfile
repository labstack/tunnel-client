FROM alpine:3.7

RUN apk add --no-cache ca-certificates

COPY dist/linux_amd64/tunnel-client /usr/local/bin/tunnel

ENTRYPOINT ["tunnel"]
