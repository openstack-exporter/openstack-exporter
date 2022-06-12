FROM golang:1.18 AS builder

WORKDIR /build
COPY . /build

RUN go mod download
RUN go build .

FROM busybox:latest AS openstack-exporter

LABEL maintainer="Jorge Niedbalski <j@bearmetal.xyz>"

COPY --from=builder /build/openstack-exporter /bin/openstack-exporter

ENTRYPOINT [ "/bin/openstack-exporter" ]
EXPOSE 9180
