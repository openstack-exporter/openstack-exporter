# Using OpenStack API microversions

OpenStack services use API microversions to expose new fields, behavior, and
resource shapes without changing the major API version. The exporter must choose
microversions deliberately because using too old a version can hide metrics, and
using too new a version can fail against older clouds.

## Current behavior

The exporter uses Gophercloud v2 service clients and configures microversions
centrally for services that advertise microversion ranges.

Service | Environment override | Default behavior
--- | --- | ---
Bare Metal (Ironic) | `OS_BAREMETAL_API_VERSION` | Uses the repo default `1.90` when supported, otherwise the cloud's advertised maximum.
Compute (Nova) | `OS_COMPUTE_API_VERSION` | Uses the cloud's advertised maximum.
Container Infra (Magnum) | `OS_CONTAINER_INFRA_API_VERSION` | Uses the cloud's advertised maximum.
Placement | `OS_PLACEMENT_API_VERSION` | Uses the cloud's advertised maximum.
Shared File Systems (Manila) | `OS_SHARE_API_VERSION` | Uses the cloud's advertised maximum.
Block Storage (Cinder) | `OS_VOLUME_API_VERSION` | Uses the cloud's advertised maximum.

When an environment variable is set, the exporter validates that the requested
microversion is supported by the cloud endpoint and uses it. When it is not set,
the exporter tries the service default first when one is configured, then falls
back to the highest microversion advertised by the cloud.

Services that do not advertise microversion ranges in Gophercloud discovery are
left unchanged.

## Operator guidance

Set the matching `OS_*_API_VERSION` variable when a cloud needs a specific API
microversion for compatibility or troubleshooting.

```sh
export OS_COMPUTE_API_VERSION=2.88
openstack-exporter --os-client-config /etc/openstack/clouds.yaml my-cloud
```

Use a value supported by the target OpenStack deployment. If the requested value
is not advertised by the Nova endpoint, the exporter fails startup for that
service instead of silently collecting partial or misleading metrics.

## Contributor guidance

Add services to the central microversion configuration when they advertise
microversion ranges and have a stable environment variable override. Avoid
setting `ServiceClient.Microversion` directly unless the service has a
documented reason that the helper cannot cover.

When adding a metric that depends on a newer field or endpoint:

* Document the minimum microversion in the relevant exporter code.
* Guard optional request behavior with `utils.IsMicroversionAtLeast`.
* Prefer graceful fallback values only when the missing field still leaves the
  metric meaningful.
* Add tests for both the older and newer response shapes when possible.

## Known Nova considerations

Nova has several metrics that depend on newer compute API behavior. For example,
hypervisor pagination is available from microversion `2.33`, and newer Nova or
Placement behavior can change where capacity data should be collected.

The exporter uses Nova discovery by default instead of carrying a hardcoded
compute microversion cap. When a deployment needs to pin older behavior, set
`OS_COMPUTE_API_VERSION` explicitly.
