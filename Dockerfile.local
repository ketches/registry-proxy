FROM alpine:latest
ARG TARGETARCH

WORKDIR /run

COPY ./bin/$TARGETARCH/registry-proxy ./registry-proxy

ENTRYPOINT ["./registry-proxy"]
