FROM golang:1.22 AS build

WORKDIR /

COPY . .

RUN go mod download && CGO_ENABLED=0 go build -o /openstack-exporter .

FROM gcr.io/distroless/base:nonroot as openstack-exporter

LABEL maintainer="Jorge Niedbalski <j@bearmetal.xyz>"

COPY --from=build /openstack-exporter /bin/openstack-exporter

ENTRYPOINT [ "/bin/openstack-exporter" ]
EXPOSE 9180
