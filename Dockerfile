FROM docker.io/library/golang:1.18-buster AS build-env

FROM build-env AS builder

WORKDIR /go/src
COPY ./ ./

# build
RUN make build

# runtime
FROM alpine:3.17.2
COPY --from=builder /go/src/srew-rot /usr/bin/srew-rot


CMD ["/usr/bin/srew-rot"]