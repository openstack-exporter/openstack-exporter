FROM quay.io/prometheus/busybox:latest
LABEL maintainer="Jorge Niedbalski <jnr@metaklass.org>"

COPY openstack_exporter /bin/openstack_exporter

ENTRYPOINT ["/bin/openstack_exporter"]
EXPOSE     9180