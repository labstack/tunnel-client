FROM alpine:3.7

COPY dist/linux_amd64/tunnel /usr/local/bin

ENTRYPOINT ["tunnel"]
