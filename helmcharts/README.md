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
container image tag.

```bash
# Install a released version (for example, exporter image 1.6.0)
helm install prometheus-openstack-exporter \
  oci://ghcr.io/openstack-exporter/charts/prometheus-openstack-exporter \
  --version 1.6.0
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
