ARG GO_VERSION=1.26.5
FROM golang:${GO_VERSION} AS build

WORKDIR /

COPY . .

RUN go mod download && CGO_ENABLED=0 go build -o /openstack-exporter .

FROM gcr.io/distroless/base:nonroot AS openstack-exporter

LABEL maintainer="Jorge Niedbalski <j@bearmetal.xyz>"

COPY --from=build /openstack-exporter /bin/openstack-exporter

ENTRYPOINT [ "/bin/openstack-exporter" ]
EXPOSE 9180
