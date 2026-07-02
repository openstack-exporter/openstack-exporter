# Using OpenStack API microversions

OpenStack services use API microversions to expose new fields, behavior, and
resource shapes without changing the major API version. The exporter must choose
microversions deliberately because using too old a version can hide metrics, and
using too new a version can fail against older clouds.

## Current behavior

The exporter uses Gophercloud v2 service clients for services that need
microversion support. For Nova, the exporter calls
`utils.SetupClientMicroversionV2` with `OS_COMPUTE_API_VERSION` and the latest
microversion supported by the exporter.

When the environment variable is set, the exporter validates that the requested
microversion is supported by the cloud endpoint and uses it. When it is not set,
the exporter tries the exporter default first, then falls back to the highest
microversion advertised by the cloud if the default is unavailable.

## Operator guidance

Set `OS_COMPUTE_API_VERSION` when a cloud needs a specific Nova API
microversion for compatibility or troubleshooting.

```sh
export OS_COMPUTE_API_VERSION=2.87
openstack-exporter --os-client-config /etc/openstack/clouds.yaml my-cloud
```

Use a value supported by the target OpenStack deployment. If the requested value
is not advertised by the Nova endpoint, the exporter fails startup for that
service instead of silently collecting partial or misleading metrics.

## Contributor guidance

Use the shared microversion helper when adding service code that depends on a
specific API microversion. Avoid setting `ServiceClient.Microversion` directly
unless the service has a documented reason that the helper cannot cover.

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

The exporter currently caps its Nova default at the latest microversion it is
known to support. Raise that default only after confirming the affected metrics
and fixtures work against the newer behavior.
