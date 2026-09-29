# Helm Chart for OpenStack Exporter

## Description

This is the official Helm Chart for [OpenStack Exporter](https://github.com/openstack-exporter/openstack-exporter), a tool to export Prometheus metrics from a running OpenStack Cloud.

## Configuration

The chart configuration is done in the `values.yaml` file.

By default the chart creates an `openstack-config` Secret from `clouds_yaml_config` and mounts it at `/etc/openstack`.
To use your own Secret instead, set `clouds_yaml_secret_name` to an existing Secret name. That Secret must contain a `clouds.yaml` key.

## Usage

Helm charts are published to GitHub Container Registry with each OpenStack
Exporter release. The chart version and application version match the released
container image tag. The image defaults to the chart application version; set
`image.tag` to override it.

```bash
# Install a released version (for example, exporter image 1.6.0)
helm install prometheus-openstack-exporter \
  oci://ghcr.io/openstack-exporter/charts/prometheus-openstack-exporter \
  --version 1.6.0
```

Every push to `main` also publishes Docker images tagged `latest` and `edge`,
and replaces the rolling Helm chart version `0.0.0-edge` with application version
`edge`. Helm OCI chart versions must be semantic versions, so the chart uses
`0.0.0-edge` rather than a literal `edge` tag.

```bash
helm upgrade --install prometheus-openstack-exporter \
  oci://ghcr.io/openstack-exporter/charts/prometheus-openstack-exporter \
  --version 0.0.0-edge
```

To render manifests for GitOps workflows such as Argo CD:

```bash
# From the repository root
helm template prometheus-openstack-exporter ./helmcharts \
  --namespace openstack \
  --set clouds_yaml_secret_name=my-openstack-config \
  > prometheus-openstack-exporter.yaml
```

Omit `--set clouds_yaml_secret_name=...` to render the chart-managed `openstack-config` Secret from `clouds_yaml_config`.

## Contributing

Please fill pull requests or issues under Github.
