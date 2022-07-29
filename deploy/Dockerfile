FROM golang AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential && \
    apt-get clean && \
    mkdir -p "$GOPATH/src/github.com/CentaurusInfra/quarkcm"

ADD . "$GOPATH/src/github.com/CentaurusInfra/quarkcm"

RUN cd "$GOPATH/src/github.com/CentaurusInfra/quarkcm" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a --installsuffix cgo --ldflags="-s" -o /quarkcm

RUN cd "$GOPATH/src/github.com/CentaurusInfra/quarkcm/cmd/quarkcni" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a --installsuffix cgo --ldflags="-s" -o /quarkcni

FROM bitnami/minideb:stretch
RUN install_packages ca-certificates

COPY --from=builder /quarkcm /bin/quarkcm
COPY --from=builder /quarkcni /bin/quarkcni

ENTRYPOINT ["/bin/quarkcm"]
