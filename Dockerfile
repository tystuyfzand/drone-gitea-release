FROM alpine

ADD drone-gitea-release /bin/

RUN apk -Uuv add ca-certificates

ENTRYPOINT ["/bin/drone-gitea-release"]