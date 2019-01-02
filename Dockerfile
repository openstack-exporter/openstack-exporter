FROM quay.io/prometheus/busybox:latest
LABEL maintainer="Jorge Niedbalski <jnr@metaklass.org>"

COPY openstack-exporter /bin/openstack-exporter

ENTRYPOINT ["/bin/openstack-exporter"]
EXPOSE     9180