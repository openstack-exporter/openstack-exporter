FROM quay.io/prometheus/busybox:latest

ARG OS=linux
ARG ARCH=amd64

LABEL maintainer="Jorge Niedbalski <jnr@metaklass.org>"

WORKDIR .build/$OS-$ARCH/openstack-exporter /bin/openstack-exporter

ENTRYPOINT ["/bin/openstack-exporter"]
EXPOSE     9180
