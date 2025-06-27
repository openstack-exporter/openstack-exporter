FROM golang:1.22 AS build

WORKDIR /

COPY . .

RUN VERSION=$(cat VERSION) && \
    REVISION=$(git rev-parse --short HEAD) && \
    BRANCH=$(git rev-parse --abbrev-ref HEAD) && \
    BUILD_USER="openstack-docker-image-githubci" && \
    BUILD_DATE=$(date -u '+%Y%m%d-%H:%M:%S') && \
    LDFLAGS="-X github.com/prometheus/common/version.Version=$VERSION -X github.com/prometheus/common/version.Revision=$REVISION -X github.com/prometheus/common/version.Branch=$BRANCH -X github.com/prometheus/common/version.BuildUser=$BUILD_USER -X github.com/prometheus/common/version.BuildDate=$BUILD_DATE" && \
    CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o /openstack-exporter .

# Check version works
RUN /openstack-exporter --version

FROM gcr.io/distroless/base:nonroot AS openstack-exporter

LABEL maintainer="Jorge Niedbalski <j@bearmetal.xyz>"

COPY --from=build /openstack-exporter /bin/openstack-exporter

ENTRYPOINT [ "/bin/openstack-exporter" ]
EXPOSE 9180
