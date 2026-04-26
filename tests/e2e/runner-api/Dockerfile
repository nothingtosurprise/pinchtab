FROM alpine:3.23

RUN apk add --no-cache \
    bash \
    curl \
    jq

WORKDIR /e2e

CMD ["/bin/bash", "/e2e/run.sh", "api", "all=true"]
