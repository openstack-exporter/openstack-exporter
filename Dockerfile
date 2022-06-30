FROM golang:1.18 AS build

WORKDIR /

COPY . .

RUN go mod download && go build -o /openstack-exporter .

FROM gcr.io/distroless/base:nonroot as openstack-exporter

LABEL maintainer="Jorge Niedbalski <j@bearmetal.xyz>"

COPY --from=build /openstack-exporter /bin/openstack-exporter

ENTRYPOINT [ "/bin/openstack-exporter" ]
EXPOSE 9180
