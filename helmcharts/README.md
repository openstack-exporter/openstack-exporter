# Helm Chart for OpenStack Exporter

## Description

This is the official Helm Chart for [OpenStack Exporter](https://github.com/openstack-exporter/openstack-exporter), a tool to export Prometheus metrics from a running OpenStack Cloud.

## Configuration

The chart configuration is done in the `values.yaml` file.

By default the chart creates an `openstack-config` Secret from `clouds_yaml_config` and mounts it at `/etc/openstack`.
To use your own Secret instead, set `clouds_yaml_secret_name` to an existing Secret name. That Secret must contain a `clouds.yaml` key.

## Usage

```bash
# Package the chart
cd charts/prometheus-openstack-exporter/
helm package .

# Get chart version & install
version="$(awk '/^version:/{ print $NF }' Chart.yaml)"
helm install prometheus-openstack-exporter prometheus-openstack-exporter-${version}.tgz
```

To render manifests for GitOps workflows such as Argo CD:

```bash
# From the repository root
helm template prometheus-openstack-exporter ./charts/prometheus-openstack-exporter \
  --namespace openstack \
  --set clouds_yaml_secret_name=my-openstack-config \
  > prometheus-openstack-exporter.yaml
```

Omit `--set clouds_yaml_secret_name=...` to render the chart-managed `openstack-config` Secret from `clouds_yaml_config`.

## Contributing

Please fill pull requests or issues under Github.
