# OpenStack Exporter for Prometheus

[![CI](https://github.com/openstack-exporter/openstack-exporter/actions/workflows/ci.yaml/badge.svg)](https://github.com/openstack-exporter/openstack-exporter/actions/workflows/ci.yaml)

A [OpenStack](https://openstack.org/) exporter for prometheus written in Golang using the
[gophercloud](https://github.com/gophercloud/gophercloud) library.

## Deployment options

The openstack-exporter can be deployed using the following mechanisms:

* By using [kolla-ansible](https://github.com/openstack/kolla-ansible) by setting enable_prometheus_openstack_exporter: true
* By using [helm charts](https://github.com/openstack-exporter/helm-charts)
* Via docker images, available from our [repository](https://github.com/openstack-exporter/openstack-exporter/pkgs/container/openstack-exporter)
* Via snaps from [snapcraft](https://snapcraft.io/golang-openstack-exporter)

### Latest Docker main images

Multi-arch images (amd64, arm64 and s390x)

```sh
docker pull ghcr.io/openstack-exporter/openstack-exporter:latest
```

### Release Docker images

Multi-arch images (amd64, arm64 and s390x)

```sh
docker pull ghcr.io/openstack-exporter/openstack-exporter:1.6.0
```

### Snaps

The exporter is also available on the [https://snapcraft.io/golang-openstack-exporter](https://snapcraft.io/golang-openstack-exporter)
For installing the latest master build (edge channel):

```sh
snap install --channel edge golang-openstack-exporter
```

For installing the latest stable version (stable channel):

```sh
snap install --channel stable golang-openstack-exporter
```

## Description

The OpenStack exporter, exports Prometheus metrics from a running OpenStack cloud
for consumption by prometheus. The cloud credentials and identity configuration
should use the [os-client-config](https://docs.openstack.org/os-client-config/latest/) format
and can be specified with the `--os-client-config` flag (defaults to `/etc/openstack/clouds.yaml`).

Other options as the binding address/port can by explored with the --help flag.

The exporter can operate in 2 modes

* A Legacy mode (targetting one cloud) in where the openstack\_exporter serves on port `0.0.0.0:9180` at the `/metrics` URL.
* A multi cloud mode in where the openstack\_exporter serves on port `0.0.0.0:9180` at the `/probe` URL.
  And where `/metrics` URL is serving own exporter metrics

You can build it by yourself by cloning this repository and run:

```sh
go build -o ./openstack-exporter .
```

Multi cloud mode

```sh
./openstack-exporter --os-client-config /etc/openstack/clouds.yaml --multi-cloud
curl "http://localhost:9180/probe?cloud=region.mycludprovider.org"
```

or Legacy mode

```sh
./openstack-exporter --os-client-config /etc/openstack/clouds.yaml myregion.cloud.org
curl "http://localhost:9180/metrics" +
```

Or alternatively you can use the docker images, as follows (check the openstack configuration section for configuration
details):

```sh
docker run -v "$HOME/.config/openstack/clouds.yml":/etc/openstack/clouds.yaml -it -p 9180:9180 \
ghcr.io/openstack-exporter/openstack-exporter:latest
curl "http://localhost:9180/probe?cloud=my-cloud.org"
```

### Command line options

The current list of command line options (by running --help)

```sh
usage: openstack-exporter [<flags>] [<cloud>]


Flags:
  -h, --[no-]help                Show context-sensitive help (also try --help-long and --help-man).
      --web.telemetry-path="/metrics"
                                 uri path to expose metrics
      --os-client-config="/etc/openstack/clouds.yaml"
                                 Path to the cloud configuration file
      --prefix="openstack"       Prefix for metrics
      --endpoint-type="public"   openstack endpoint type to use (i.e: public, internal, admin)
      --[no-]collect-metric-time
                                 time spent collecting each metric
  -d, --disable-metric= ...      multiple --disable-metric can be specified in the format: service-metric (i.e: cinder-snapshots)
      --[no-]disable-slow-metrics
                                 Disable slow metrics for performance reasons
      --[no-]disable-deprecated-metrics
                                 Disable deprecated metrics
      --[no-]disable-cinder-agent-uuid
                                 Disable UUID generation for Cinder agents
      --[no-]multi-cloud         Toggle the multiple cloud scraping mode under /probe?cloud=
      --domain-id=DOMAIN-ID      Gather metrics only for the given Domain ID (defaults to all domains)
      --[no-]cache               Enable Cache mechanism globally
      --cache-ttl=300s           TTL duration for cache expiry(eg. 10s, 11m, 1h)

      --[no-]disable-service.network
                                 Disable the network service exporter
      --[no-]disable-service.compute
                                 Disable the compute service exporter
      --[no-]disable-service.image
                                 Disable the image service exporter
      --[no-]disable-service.volume
                                 Disable the volume service exporter
      --[no-]disable-service.identity
                                 Disable the identity service exporter
      --[no-]disable-service.object-store
                                 Disable the object-store service exporter
      --[no-]disable-service.load-balancer
                                 Disable the load-balancer service exporter
      --[no-]disable-service.container-infra
                                 Disable the container-infra service exporter
      --[no-]disable-service.dns
                                 Disable the dns service exporter
      --[no-]disable-service.baremetal
                                 Disable the baremetal service exporter
      --[no-]disable-service.gnocchi
                                 Disable the gnocchi service exporter
      --[no-]disable-service.database
                                 Disable the database service exporter
      --[no-]disable-service.orchestration
                                 Disable the orchestration service exporter
      --[no-]disable-service.placement
                                 Disable the placement service exporter
      --[no-]disable-service.sharev2  
                                 Disable the share service exporter
      --[no-]web.systemd-socket  Use systemd socket activation listeners instead of port listeners (Linux only).
      --web.listen-address=:9180 ...
                                 Addresses on which to expose metrics and web interface. Repeatable for multiple addresses.
      --web.config.file=""       [EXPERIMENTAL] Path to configuration file that can enable TLS or authentication. See:
                                 https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md
      --log.level=info           Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt, json]
      --[no-]version             Show application version.

Args:
  [<cloud>]  name or id of the cloud to gather metrics from
```

### Scrape options

In legacy mode cloud and metrics to be scraped are specified as argument or flags as described above.
To select multi cloud mode the --multi-cloud flag needs to be used.
In that case metrics and clouds are specified in the http scrape request as described below.
Which cloud (name or id from the `clouds.yaml` file) or what services from the cloud to scrape, can be specified as the parameters to http scrape requests.

Query Parameter | Description
--- | ---
`cloud` | Name or id of the cloud to gather metrics from (as specified in the `clouds.yaml`)
`include_services` | A comma separated list of services for which metrics will be scraped. It ignores flags for disabling services `--disable-service.*`.
`exclude_services` | A comma separated list of services for which metrics will *not* be scraped. Default is empty: ""

#### Examples

Scrape all services from `test.cloud`:

```sh
curl "https://localhost:9180/probe?cloud=test.cloud"
```

Scrape only `network` and `compute` services from `test.cloud`:

```sh
curl "https://localhost:9180/probe?cloud=test.cloud&include_services=network,compute"
```

Scrape all services except `load-balancer` and `dns` from `test.cloud`:

```sh
curl "https://localhost:9180/probe?cloud=test.cloud&exclude_services=load-balancer,dns"
```

### OpenStack configuration

The cloud credentials and identity configuration
should use the [os-client-config](https://docs.openstack.org/os-client-config/latest/) format
and can be specified with the `--os-client-config` flag (defaults to `/etc/openstack/clouds.yaml`).

`cacert` can be a full path to a PEM encoded file or contents of a PEM encoded file.

Current list of supported options can be seen in the following example
configuration:

```yaml
clouds:
  default:
    region_name: {{ openstack_region_name }}
    identity_api_version: 3
    identity_interface: internal
    auth:
      username: {{ keystone_admin_user }}
      password: {{ keystone_admin_password }}
      project_name: {{ keystone_admin_project }}
      project_domain_name: 'Default'
      project_domain_id: 'Default' // This can replace "project_domain_name"
      user_domain_name: 'Default'
      auth_url: {{ admin_protocol }}://{{ kolla_internal_fqdn }}:{{ keystone_admin_port }}/v3
    cacert: |
      ---- BEGIN CERTIFICATE ---
      ...
    verify: true | false  // disable || enable SSL certificate verification
```

### OpenStack Domain filtering

The exporter provides the flag `--domain-id`, this restricts some metrics to a specific domain.

*Restricting domain scope can improve scrape time, especially if you use Heat a lot.*

The following metrics are filtered for the domain ID provided (the others remain the same):

#### Cinder

* `openstack_cinder_limits_volume_max_gb`
* `openstack_cinder_limits_volume_used_gb`
* `openstack_cinder_limits_backup_max_gb`
* `openstack_cinder_limits_backup_used_gb`

#### Keystone

* `openstack_identity_projects`
* `openstack_identity_project_info`

#### Nova

* `openstack_nova_limits_vcpus_max`
* `openstack_nova_limits_vcpus_used`
* `openstack_nova_limits_memory_max`
* `openstack_nova_limits_memory_used`
* `openstack_nova_limits_instances_max`
* `openstack_nova_limits_instances_used`

### Cache mechanism

Enabling the cache with `--cache` changes the exporter's metric collection and delivery:

#### Background Service

* Collects metrics at the start and subsequently every half cache TTL.
* Updates the cache backend after completing each collection cycle.
* Flushes expired cache data every cache TTL.

#### Exporter API

* Returns no data if the cache is empty or expired.
* Retrieves and returns cached data from the backend.

## Contributing

Please file pull requests or issues under GitHub. Feel free to request any metrics
that might be missing.

### Operational Concerns

#### OpenStack Exporter Compatibility with Older OpenStack Versions

OpenStack evolves with new features and fields added to the API over time, leveraging a system of microversions to allow for incremental changes while maintaining backward compatibility. However, not all OpenStack deployments will be running the latest microversions, meaning certain fields or metrics may not be available when using an older version of OpenStack. Similarly, as newer OpenStack versions are adopted with more recent microversions, these changes are also seamlessly handled.

OpenStack-Exporter is designed to handle both older and newer microversions gracefully by default. When querying an OpenStack environment, it ensures compatibility with various microversions, using fallback behaviors for missing fields introduced in newer microversions:

* For Boolean Fields: Assumes the default value of false.
* For Numeric Fields: Missing numeric fields will default to 0
* For String Fields: Typically default to an empty string ("")

This fallback mechanism ensures that OpenStack-Exporter works correctly even when interfacing with OpenStack environments using older microversions, without causing operational disruptions.

## Metrics

### Slow metrics

There are some metrics that, depending on the cloud deployment size, can be slow to be
collected because iteration over different projects is required. Those metrics are marked as `slow` and can be disabled with the command
line parameter `--disable-slow-metrics`.

Currently flagged as slow metrics are:

Name | Exporter
-----|------------
limits_vcpus_max | nova
limits_vcpus_used | nova
limits_memory_max | nova
limits_memory_used | nova
limits_instances_max | nova
limits_instances_used | nova
limits_volume_max_gb | cinder
limits_volume_used_gb |  cinder
limits_backup_max_gb | cinder
limits_backup_used_gb | cinder
image_bytes | glance
image_created_at | glance

#### Deprecated Metrics

Metric name |  Since Version | Removed in Version | Notes
------------|------------|--------------|-------------------------------------
openstack_cinder_volume_status | 1.4 | 1.5 | deprecated in favor of openstack_cinder_volume_gb

#### Metrics collected

Name     | Sample Labels                                                                                                                                                                                                                                                                                                         | Sample Value | Description
---------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------|------------
openstack_glance_image_bytes| id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"                                                                                                                                                                                                |1.3167616e+07 (float)| Image size in bytes
openstack_glance_image_created_at| hidden="false",id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"                                                                                                                                                                       | 1.415380026e+09| Image creation timestamp
openstack_glance_images| region="Region"                                                                                                                                                                                                                                                                                                       |1.0 (float)| Total number of images
openstack_neutron_agent_state| adminState="up",availability_zone="nova",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"                                                                                                                                                                                                        |1 or 0 (bool)| Agent state (1=up, 0=down)
openstack_neutron_floating_ip| region="RegionOne",floating_ip_address="172.24.4.227",floating_network_id="1c93472c-4d8a-11ea-92e9-08002759fd91",id="231facca-4d8a-11ea-a143-08002759fd91",project_id="0042b7564d8a11eabc2d08002759fd91",router_id="",status="DOWN"                                                                                   |4.0 (float)| Floating IP status
openstack_neutron_floating_ips| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of floating IPs
openstack_neutron_networks| region="RegionOne"                                                                                                                                                                                                                                                                                                    |25.0 (float)| Total number of networks
openstack_neutron_ports| region="RegionOne"                                                                                                                                                                                                                                                                                                    | 1063.0 (float)| Total number of ports
openstack_neutron_subnets| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of subnets
openstack_neutron_subnets_total| ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"                                                                                                                    |8 (float)| Total subnets in pool
openstack_neutron_subnets_used| ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"                                                                                                                    |1 (float)| Used subnets in pool
openstack_neutron_subnets_free| ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"                                                                                                                    |7 (float)| Free subnets in pool
openstack_neutron_security_groups| region="RegionOne"                                                                                                                                                                                                                                                                                                    |10.0 (float)| Total number of security groups
openstack_neutron_network_ip_availabilities_total| region="RegionOne",network_id="23046ac4-67fc-4bf6-842b-875880019947",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="478340c7c6bf49c99ce40641fd13ba96"                                                                                                                          |253.0 (float)| Total available IPs in network
openstack_neutron_network_ip_availabilities_used| region="RegionOne",network_id="23046ac4-67fc-4bf6-842b-875880019947",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="478340c7c6bf49c99ce40641fd13ba96"                                                                                                                          |151.0 (float)| Used IPs in network
openstack_neutron_router| admin_state_up="true",external_network_id="78620e54-9ec2-4372-8b07-3ac2d02e0288",id="9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f",name="router2",project_id="a2a651cc26974de98c9a1f9aa88eb2e6",status="N/A"                                                                                                                  | 1.0 (float)| Router information
openstack_neutron_routers| region="RegionOne"                                                                                                                                                                                                                                                                                                    |134.0 (float)| Total number of routers
openstack_neutron_l3_agent_of_router| region="RegionOne",agent_admin_up="true",agent_alive="true",agent_host="dev-os-ctrl-02",ha_state="",l3_agent_id="ddbf087c-e38f-4a73-bcb3-c38f2a719a03",router_id="9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f"                                                                                                               |1.0 (float)| L3 agent router assignment
openstack_neutron_network | id="d32019d3-bc6e-4319-9c1d-6722fc136a22",is_external="false",is_shared="false",name="net1",provider_network_type="vlan",provider_physical_network="public",provider_segmentation_id="3",status="ACTIVE",subnets="54d6f61d-db07-451c-9ab3-b9609b6b6f0b",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869" | 1 (float)| Network information
openstack_neutron_subnet | cidr="10.10.0.0/24",dns_nameservers="",enable_dhcp="true",gateway_ip="10.10.0.1",id="12769bb8-6c3c-11ec-8124-002b67875abf",name="pooled-subnet-ipv4",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"                                                 | 1 (float)| Subnet information
openstack_loadbalancer_up |                                                                                                                                                                                                                                                                                                                       | 1 (float)| Load balancer service status
openstack_loadbalancer_total_loadbalancers|                                                                                                                                                                                                                                                                                                                       | 2 (float)| Total number of load balancers
openstack_loadbalancer_loadbalancer_status | id="607226db-27ef-4d41-ae89-f2a800e9c2db",name="best_load_balancer",operating_status="ONLINE",project_id="e3cd678b11784734bc366148aa37580e",provider="octavia",provisioning_status="ACTIVE",vip_address="203.0.113.50"                                                                                                | 0 (float)| Load balancer status
openstack_loadbalancer_total_amphorae|                                                                                                                                                                                                                                                                                                                       | 2 (float)| Total number of amphorae
openstack_loadbalancer_amphora_status| cert_expiration="2020-08-08T23:44:31Z",compute_id="667bb225-69aa-44b1-8908-694dc624c267",ha_ip="10.0.0.6",id="45f40289-0551-483a-b089-47214bc2a8a4",lb_network_ip="192.168.0.6",loadbalancer_id="882f2a9d-9d53-4bd0-b0e9-08e9d0de11f9",role="MASTER",status="READY"                                                   | 2.0 (float)| Amphora status
openstack_loadbalancer_total_pools|                                                                                                                                                                                                                                                                                                                       | 2 (float)| Total number of pools
openstack_loadbalancer_pool_status| id="ca00ed86-94e3-440e-95c6-ffa35531081e",lb_algorithm="ROUND_ROBIN",loadbalancers="e7284bb2-f46a-42ca-8c9b-e08671255125",name="my_test_pool",operating_status="ERROR",project_id="8b1632d90bfe407787d9996b7f662fd7",protocol="TCP",provisioning_status="ACTIVE"                                                   | 2.0 (float)| Pool status
openstack_nova_availability_zones| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of availability zones
openstack_nova_flavor| disk="disk",id="id",is_public="is_public",name="name",ram="ram",vcpus="vcpus"                                                                                                                                                                                                                                                     |1.0 (float)| Flavor information
openstack_nova_flavors| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of flavors
openstack_nova_total_vms| region="RegionOne"                                                                                                                                                                                                                                                                                                    |12.0 (float)| Total number of VMs
openstack_nova_server_status| region="RegionOne",hostname="compute-01",id="id",name="name",tenant_id="tenant_id",user_id="user_id",address_ipv4="address_ipv4",address_ipv6="address_ipv6",host_id="host_id",uuid="uuid",availability_zone="availability_zone"                                                                                             |0.0 (float)| Server status
openstack_nova_running_vms| region="RegionOne",hostname="compute-01",availability_zone="az1",aggregates="shared,ssd"                                                                                                                                                                                                                              |12.0 (float)| Number of running VMs
openstack_nova_server_local_gb| id="27bb2854-b06a-48f5-ab4e-139817b8b8ff",name="openstack-monitoring-0",tenant_id="110f6313d2d346b4aa90eabe4970b62a"                                                                                                                                                                                                 | 10 (float)| Server local disk size
openstack_nova_free_disk_bytes| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |1230.0 (float)| Free disk space in bytes
openstack_nova_local_storage_used_bytes| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |100.0 (float)| Used local storage in bytes
openstack_nova_local_storage_available_bytes| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |30.0 (float)| Available local storage in bytes
openstack_nova_memory_used_bytes| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |40000.0 (float)| Used memory in bytes
openstack_nova_memory_available_bytes| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |40000.0 (float)| Available memory in bytes
openstack_nova_agent_state| hostname="compute-01",region="RegionOne",id="288",service="nova-compute",adminState="enabled",zone="nova"                                                                                                                                                                                                           |1.0 or 0 (bool)| Agent state (1=up, 0=down)
openstack_nova_vcpus_available| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |128.0 (float)| Available vCPUs
openstack_nova_vcpus_used| region="RegionOne",hostname="compute-01",aggregates="shared,ssd"                                                                                                                                                                                                                                                      |32.0 (float)| Used vCPUs
openstack_nova_limits_vcpus_max| tenant="demo-project"                                                                                                                                                                                                                                                                                                 |128.0 (float)| Maximum vCPUs limit
openstack_nova_limits_vcpus_used| tenant="demo-project"                                                                                                                                                                                                                                                                                                 |32.0 (float)| Used vCPUs count
openstack_nova_limits_memory_max| tenant="demo-project"                                                                                                                                                                                                                                                                                                 |40000.0 (float)| Maximum memory limit
openstack_nova_limits_memory_used| tenant="demo-project"                                                                                                                                                                                                                                                                                                 |40000.0 (float)| Used memory count
openstack_nova_limits_instances_max| tenant="demo-project"                                                                                                                                                                                                                                                                                                 |15.0 (float)| Maximum instances limit
openstack_nova_limits_instances_used| tenant="demo-project"                                                                                                                                                                                                                                                                                                 |5.0 (float)| Used instances count
openstack_cinder_service_state| hostname="compute-01",region="RegionOne",service="cinder-backup",adminState="enabled",zone="nova"                                                                                                                                                                                                                     |1.0 or 0 (bool)| Service state (1=up, 0=down)
openstack_cinder_limits_volume_max_gb| tenant="demo-project",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"                                                                                                                                                                                                                                                    |40000.0 (float)| Maximum volume size limit
openstack_cinder_limits_volume_used_gb| tenant="demo-project",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"                                                                                                                                                                                                                                                    |40000.0 (float)| Used volume size
openstack_cinder_limits_backup_max_gb| tenant="demo-project",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"                                                                                                                                                                                                                                                    |1000.0 (float)| Maximum backup size limit
openstack_cinder_limits_backup_used_gb| tenant="demo-project",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"                                                                                                                                                                                                                                                    |0.0 (float)| Used backup size
openstack_cinder_volumes| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of volumes
openstack_cinder_snapshots| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of snapshots
openstack_cinder_volume_status| region="RegionOne",bootable="true",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",size="1",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1",server_id="f4fda93b-06e0-4743-8117-bc8bcecd651b"                                                                   |4.0 (float)| Volume status
openstack_cinder_volume_gb| region="RegionOne",availability_zone="nova",bootable="true",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",user_id="32779452fcd34ae1a53a797ac8a1e064",volume_type="lvmdriver-1",server_id="f4fda93b-06e0-4743-8117-bc8bcecd651b"        |4.0 (float)| Volume size in GB
openstack_designate_zones| region="RegionOne"                                                                                                                                                                                                                                                                                                    |4.0 (float)| Total number of DNS zones
openstack_designate_zone_status| region="RegionOne",id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",name="example.org.",status="ACTIVE",tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",type="PRIMARY"                                                                                                                                                      |4.0 (float)| DNS zone status
openstack_designate_recordsets| region="RegionOne",tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",zone_id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",zone_name="example.org."                                                                                                                                                                           |4.0 (float)| Total number of recordsets
openstack_designate_recordsets_status| region="RegionOne",id="f7b10e9b-0cae-4a91-b162-562bc6096648",name="example.org.",status="PENDING",type="A",zone_id="2150b1bf-dee2-4221-9d85-11f7886fb15f",zone_name="example.com."                                                                                                                                    |4.0 (float)| Recordset status
openstack_identity_domains| region="RegionOne"                                                                                                                                                                                                                                                                                                    |1.0 (float)| Total number of domains
openstack_identity_users| region="RegionOne"                                                                                                                                                                                                                                                                                                    |30.0 (float)| Total number of users
openstack_identity_projects| region="RegionOne"                                                                                                                                                                                                                                                                                                    |33.0 (float)| Total number of projects
openstack_identity_project_info| is_domain="false",description="This is a project description",domain_id="default",enabled="true",id="0c4e939acacf4376bdcd1129f1a054ad",name="demo-project",parent_id=""                                                                                                                                                |1.0 (float)| Project information
openstack_identity_groups| region="RegionOne"                                                                                                                                                                                                                                                                                                    |1.0 (float)| Total number of groups
openstack_identity_regions| region="RegionOne"                                                                                                                                                                                                                                                                                                    |1.0 (float)| Total number of regions
openstack_object_store_objects| region="RegionOne",container_name="test2"                                                                                                                                                                                                                                                                             |1.0 (float)| Number of objects in container
openstack_object_store_bytes|region="RegionOne",container_name="test2"                                                                                                                                                                                                                                                                             |1.0 (float) | Object bytes in a container
openstack_container_infra_cluster_masters| name="k8s",node_count="1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"                                                                                                                            |1 (float)| Number of cluster master nodes
openstack_container_infra_cluster_nodes| master_count="1",name="k8s",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"                                                                                                                          |1 (float)| Number of cluster worker nodes
openstack_container_infra_cluster_status| master_count="1",name="k8s",node_count="1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"                                                                                                           |1 (float)| Cluster status
openstack_trove_instance_status| datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"                                                                                                      |2 (float)| Database instance status
openstack_trove_instance_volume_size_gb| datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"                                                                                                      |20 (float)| Database instance volume size
openstack_trove_instance_volume_used_gb| datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"                                                                                                      |0.4 (float)| Database instance volume used
openstack_heat_stack_status| id="00cb0780-c883-4964-89c3-b79d840b3cbf",name="demo-stack2",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="CREATE_COMPLETE"                                                                                                                                                                                   |5 (float)| Heat stack status
openstack_heat_stack_status_counter| status="CREATE_COMPLETE"                                                                                                                                                                                                                                                                                              |1 (float)| Heat stack status counter
openstack_placement_resource_allocation_ratio| hostname="compute-01",resourcetype="DISK_GB\|PCPU\|VCPU\|..."                                                                                                                                                                                                                                                           |1.2 (float)| Resource allocation ratio
openstack_placement_resource_reserved| hostname="compute-01",resourcetype="DISK_GB\|PCPU\|VCPU\|..."                                                                                                                                                                                                                                                           |8 (float)| Reserved resources
openstack_placement_resource_total| hostname="compute-01",resourcetype="DISK_GB\|PCPU\|VCPU\|..."                                                                                                                                                                                                                                                           |80 (float)| Total resources
openstack_placement_resource_usage| hostname="compute-01",resourcetype="DISK_GB\|PCPU\|VCPU\|..."                                                                                                                                                                                                                                                           |40 (float)| Used resources
openstack_metric_collect_seconds | openstack_metric="agent_state",openstack_service="openstack_cinder"                                                                                                                                                                                                                                                 |1.27843913| Metric collection time (only if --collect-metric-time is passed)

## Cinder Volume Status Description

Index | Status
------|-------
0 |creating
1 |available
2 |reserved
3 |attaching
4 |detaching
5 |in-use
6 |maintenance
7 |deleting
8 |awaiting-transfer
9 |error
10 |error_deleting
11 |backing-up
12 |restoring-backup
13 |error_backing-up
14 |error_restoring
15 |error_extending
16 |downloading
17 |uploading
18 |retyping
19 |extending

## Manila Share Status Description

Index | Status
------|-------
0 |creating
1 |available
2 |updating
3 |migrating
4 |migration_error
5 |extending
6 |deleting
7 |shrinking
8 |error
9 |error_deleting
10 |shrinking_error
11 |reverting_error
12 |restoring-backup
13 |restoring
14 |reverting
15 |managing
16 |unmanaging
17 |reverting_to_snapshot
18 |soft_deleting
19 |inactive

## Example metrics

```text
# HELP openstack_cinder_agent_state agent_state
# TYPE openstack_cinder_agent_state counter
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-01",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-01",region="Region",service="cinder-scheduler",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-01@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-02",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-02",region="Region",service="cinder-scheduler",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-02@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-03",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-03",region="Region",service="cinder-scheduler",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-03@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-04",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-04@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-05",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-05@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-06",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-06@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-07",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-07@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-09",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-09@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-10",region="Region",service="cinder-backup",zone="nova"} 1.0
openstack_cinder_agent_state{adminState="enabled",hostname="compute-node-10@rbd-1",region="Region",service="cinder-volume",zone="nova"} 1.0
# HELP openstack_cinder_volume_status volume_status
# TYPE openstack_cinder_volume_status gauge
openstack_cinder_volume_status{bootable="false",id="6edbc2f4-1507-44f8-ac0d-eed1d2608d38",name="test-volume-attachments",server_id="f4fda93b-06e0-4743-8117-bc8bcecd651b",size="2",status="in-use",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1"} 5
openstack_cinder_volume_status{bootable="true",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",server_id="",size="1",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1"} 1
# HELP openstack_cinder_volume_gb volume_gb
# TYPE openstack_cinder_volume_gb gauge
openstack_cinder_volume_gb{availability_zone="nova",bootable="false",id="6edbc2f4-1507-44f8-ac0d-eed1d2608d38",name="test-volume-attachments",server_id="f4fda93b-06e0-4743-8117-bc8bcecd651b",status="in-use",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",user_id="32779452fcd34ae1a53a797ac8a1e064",volume_type="lvmdriver-1"} 2
openstack_cinder_volume_gb{availability_zone="nova",bootable="true",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",server_id="",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",user_id="32779452fcd34ae1a53a797ac8a1e064",volume_type="lvmdriver-1"} 1
# HELP openstack_cinder_limits_backup_max_gb limits_backup_max_gb
# TYPE openstack_cinder_limits_backup_max_gb gauge
openstack_cinder_limits_backup_max_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 1000
openstack_cinder_limits_backup_max_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 1000
openstack_cinder_limits_backup_max_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 1000
openstack_cinder_limits_backup_max_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 1000
openstack_cinder_limits_backup_max_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 1000
openstack_cinder_limits_backup_max_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 1000
openstack_cinder_limits_backup_max_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 1000
openstack_cinder_limits_backup_max_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 1000
# HELP openstack_cinder_limits_backup_used_gb limits_backup_used_gb
# TYPE openstack_cinder_limits_backup_used_gb gauge
openstack_cinder_limits_backup_used_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_backup_used_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_backup_used_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_backup_used_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_backup_used_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_backup_used_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_backup_used_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_backup_used_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_limits_volume_max_gb limits_volume_max_gb
# TYPE openstack_cinder_limits_volume_max_gb gauge
openstack_cinder_limits_volume_max_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 1000
openstack_cinder_limits_volume_max_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 1000
openstack_cinder_limits_volume_max_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 1000
openstack_cinder_limits_volume_max_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 1000
openstack_cinder_limits_volume_max_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 1000
openstack_cinder_limits_volume_max_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 1000
openstack_cinder_limits_volume_max_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 1000
openstack_cinder_limits_volume_max_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 1000
# HELP openstack_cinder_limits_volume_used_gb limits_volume_used_gb
# TYPE openstack_cinder_limits_volume_used_gb gauge
openstack_cinder_limits_volume_used_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_volume_used_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_volume_used_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_volume_used_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_volume_used_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_volume_used_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_volume_used_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_volume_used_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_snapshots snapshots
# TYPE openstack_cinder_snapshots gauge
openstack_cinder_snapshots{region="Region"} 0.0
# HELP openstack_cinder_volumes volumes
# TYPE openstack_cinder_volumes gauge
openstack_cinder_volumes{region="Region"} 8.0
# HELP openstack_designate_recordsets recordsets
# TYPE openstack_designate_recordsets gauge
openstack_designate_recordsets{tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",zone_id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",zone_name="example.org."} 1
# HELP openstack_designate_recordsets_status recordsets_status
# TYPE openstack_designate_recordsets_status gauge
openstack_designate_recordsets_status{id="f7b10e9b-0cae-4a91-b162-562bc6096648",name="example.org.",status="PENDING",type="A",zone_id="2150b1bf-dee2-4221-9d85-11f7886fb15f",zone_name="example.com."} 0
# HELP openstack_designate_up up
# TYPE openstack_designate_up gauge
openstack_designate_up 1
# HELP openstack_designate_zone_status zone_status
# TYPE openstack_designate_zone_status gauge
openstack_designate_zone_status{id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",name="example.org.",status="ACTIVE",tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",type="PRIMARY"} 1
# HELP openstack_designate_zones zones
# TYPE openstack_designate_zones gauge
openstack_designate_zones 1
# HELP openstack_container_infra_cluster_masters cluster_masters
# TYPE openstack_container_infra_cluster_masters gauge
openstack_container_infra_cluster_masters{name="k8s",node_count="1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_cluster_nodes cluster_nodes
# TYPE openstack_container_infra_cluster_nodes gauge
openstack_container_infra_cluster_nodes{master_count="1",name="k8s",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_cluster_status cluster_status
# TYPE openstack_container_infra_cluster_status gauge
openstack_container_infra_cluster_status{master_count="1",name="k8s",node_count="1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_total_clusters total_clusters
# TYPE openstack_container_infra_total_clusters gauge
openstack_container_infra_total_clusters 1
# HELP openstack_container_infra_up up
# TYPE openstack_container_infra_up gauge
openstack_container_infra_up 1
# HELP openstack_glance_image_bytes image_bytes
# TYPE openstack_glance_image_bytes gauge
openstack_glance_image_bytes{id="781b3762-9469-4cec-b58d-3349e5de4e9c",name="F17-x86_64-cfntools",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"} 4.76704768e+08
openstack_glance_image_bytes{id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"} 1.3167616e+07
# HELP openstack_glance_image_created_at image_created_at
# TYPE openstack_glance_image_created_at gauge
openstack_glance_image_created_at{hidden="false",id="781b3762-9469-4cec-b58d-3349e5de4e9c",name="F17-x86_64-cfntools",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"} 1.414657419e+09
openstack_glance_image_created_at{hidden="false",id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"} 1.415380026e+09
# HELP openstack_glance_images images
# TYPE openstack_glance_images gauge
openstack_glance_images{region="Region"} 18.0
# HELP openstack_gnocchi_status_measures_to_process status_measures_to_process
# TYPE openstack_gnocchi_status_measures_to_process gauge
openstack_gnocchi_status_measures_to_process 291
# HELP openstack_gnocchi_status_metric_having_measures_to_process status_metric_having_measures_to_process
# TYPE openstack_gnocchi_status_metric_having_measures_to_process gauge
openstack_gnocchi_status_metric_having_measures_to_process 291
# HELP openstack_gnocchi_status_metricd_processors status_metricd_processors
# TYPE openstack_gnocchi_status_metricd_processors gauge
openstack_gnocchi_status_metricd_processors 8
# HELP openstack_gnocchi_total_metrics total_metrics
# TYPE openstack_gnocchi_total_metrics gauge
openstack_gnocchi_total_metrics 2759
# HELP openstack_identity_domains domains
# TYPE openstack_identity_domains gauge
openstack_identity_domains 1
# HELP openstack_identity_groups groups
# TYPE openstack_identity_groups gauge
openstack_identity_groups 2
# HELP openstack_identity_project_info project_info
# TYPE openstack_identity_project_info gauge
openstack_identity_project_info{description="",domain_id="1bc2169ca88e4cdaaba46d4c15390b65",enabled="true",id="4b1eb781a47440acb8af9850103e537f",is_domain="false",name="swifttenanttest4",parent_id=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="0c4e939acacf4376bdcd1129f1a054ad",is_domain="false",name="admin",parent_id=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="2db68fed84324f29bb73130c6c2094fb",is_domain="false",name="swifttenanttest2",parent_id=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="3d594eb0f04741069dbbb521635b21c7",is_domain="false",name="service",parent_id=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="43ebde53fc314b1c9ea2b8c5dc744927",is_domain="false",name="swifttenanttest1",parent_id=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="5961c443439d4fcebe42643723755e9d",is_domain="false",name="invisible_to_admin",parent_id=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="fdb8424c4e4f4c0ba32c52e2de3bd80e",is_domain="false",name="alt_demo",parent_id=""} 1
openstack_identity_project_info{description="This is a demo project.",domain_id="default",enabled="true",id="0cbd49cbf76d405d9c86562e1d579bd3",is_domain="false",name="demo",parent_id=""} 1
# HELP openstack_identity_projects projects
# TYPE openstack_identity_projects gauge
openstack_identity_projects 8
# HELP openstack_identity_regions regions
# TYPE openstack_identity_regions gauge
openstack_identity_regions 1
# HELP openstack_identity_up up
# TYPE openstack_identity_up gauge
openstack_identity_up 1
# HELP openstack_identity_users users
# TYPE openstack_identity_users gauge
openstack_identity_users 2
# HELP openstack_ironic_node node
# TYPE openstack_ironic_node gauge
openstack_ironic_node{console_enabled="true",id="f6965a47-324f-41fa-995e-0011333aa79e",maintenance="false",name="r1-02",power_state="power off",provision_state="available"} 1
openstack_ironic_node{console_enabled="true",id="a016f9c9-3faf-425b-88a4-a16e4308d72d",maintenance="false",name="r1-04",power_state="power off",provision_state="available"} 1
openstack_ironic_node{console_enabled="true",id="0fbd1d8c-2842-4d90-b1e0-43e13c195fd5",maintenance="false",name="r1-05",power_state="power off",provision_state="available"} 1
openstack_ironic_node{console_enabled="true",id="3fc2e062-7826-46ec-8bd1-695511e30a0c",maintenance="false",name="r1-03",power_state="power off",provision_state="available"} 1
openstack_ironic_node{console_enabled="true",id="b3d57927-206f-4eed-97a2-33069c12efa7",maintenance="false",name="r1-01",power_state="power off",provision_state="available"} 1
# HELP openstack_ironic_up up
# TYPE openstack_ironic_up gauge
openstack_ironic_up 1
# HELP openstack_neutron_agent_state agent_state
# TYPE openstack_neutron_agent_state counter
openstack_neutron_agent_state{adminState="up",hostname="compute-node-01",region="Region",service="neutron-dhcp-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-01",region="Region",service="neutron-l3-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-01",region="Region",service="neutron-metadata-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-01",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-02",region="Region",service="neutron-dhcp-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-02",region="Region",service="neutron-l3-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-02",region="Region",service="neutron-metadata-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-02",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-03",region="Region",service="neutron-dhcp-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-03",region="Region",service="neutron-l3-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-03",region="Region",service="neutron-metadata-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-03",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-04",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-05",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-06",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-07",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-09",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-10",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-01",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-02",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-03",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-04",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-05",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-07",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-08",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-09",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-10",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-11",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-12",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-13",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-15",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-17",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-18",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-19",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-20",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-21",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-22",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-23",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-24",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-25",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-26",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-27",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-28",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-29",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-31",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-32",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-34",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-35",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-36",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-37",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-38",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-39",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-40",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-42",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-43",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-44",region="Region",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-node-extra-45",region="Region",service="neutron-openvswitch-agent"} 1.0
# HELP openstack_neutron_floating_ip floating_ip
# TYPE openstack_neutron_floating_ip gauge
openstack_neutron_floating_ip{floating_ip_address="172.24.4.227",floating_network_id="1c93472c-4d8a-11ea-92e9-08002759fd91",id="231facca-4d8a-11ea-a143-08002759fd91",project_id="0042b7564d8a11eabc2d08002759fd91",router_id="",status="DOWN"} 1
openstack_neutron_floating_ip{floating_ip_address="172.24.4.227",floating_network_id="376da547-b977-4cfe-9cba-275c80debf57",id="61cea855-49cb-4846-997d-801b70c71bdd",project_id="4969c491a3c74ee4af974e6d800c62de",router_id="",status="DOWN"} 1
openstack_neutron_floating_ip{floating_ip_address="172.24.4.228",floating_network_id="376da547-b977-4cfe-9cba-275c80debf57",id="2f245a7b-796b-4f26-9cf9-9e82d248fda7",project_id="4969c491a3c74ee4af974e6d800c62de",router_id="d23abc8d-2991-4a55-ba98-2aaea84cc72f",status="ACTIVE"} 1
openstack_neutron_floating_ip{floating_ip_address="172.24.4.42",floating_network_id="376da547-b977-4cfe-9cba-275c80debf57",id="898b198e-49f7-47d6-a7e1-53f626a548e6",project_id="4969c491a3c74ee4af974e6d800c62de",router_id="0303bf18-2c52-479c-bd68-e0ad712a1639",status="ACTIVE"} 1
# HELP openstack_neutron_floating_ips floating_ips
# TYPE openstack_neutron_floating_ips gauge
openstack_neutron_floating_ips{region="Region"} 22.0
# HELP openstack_neutron_floating_ips_associated_not_active floating_ips_associated_not_active
# TYPE openstack_neutron_floating_ips_associated_not_active gauge
openstack_neutron_floating_ips_associated_not_active 1
# HELP openstack_neutron_l3_agent_of_router l3_agent_of_router
# TYPE openstack_neutron_l3_agent_of_router gauge
openstack_neutron_l3_agent_of_router{agent_admin_up="true",agent_alive="true",agent_host="dev-os-ctrl-02",ha_state="",l3_agent_id="ddbf087c-e38f-4a73-bcb3-c38f2a719a03",router_id="9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f"} 1
openstack_neutron_l3_agent_of_router{agent_admin_up="true",agent_alive="true",agent_host="dev-os-ctrl-02",ha_state="",l3_agent_id="ddbf087c-e38f-4a73-bcb3-c38f2a719a03",router_id="f8a44de0-fc8e-45df-93c7-f79bf3b01c95"} 1
# HELP openstack_neutron_network network
# TYPE openstack_neutron_network gauge
openstack_neutron_network{id="d32019d3-bc6e-4319-9c1d-6722fc136a22",is_external="false",is_shared="false",name="net1",provider_network_type="vlan",provider_physical_network="public",provider_segmentation_id="3",status="ACTIVE",subnets="54d6f61d-db07-451c-9ab3-b9609b6b6f0b",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 0
openstack_neutron_network{id="db193ab3-96e3-4cb3-8fc5-05f4296d0324",is_external="false",is_shared="false",name="net2",provider_network_type="local",provider_physical_network="",provider_segmentation_id="",status="ACTIVE",subnets="08eae331-0402-425a-923c-34f7cfe39c1b",tags="tag1,tag2",tenant_id="26a7980765d0414dbc1fc1f88cdb7e6e"} 0
# HELP openstack_neutron_networks networks
# TYPE openstack_neutron_networks gauge
openstack_neutron_networks{region="Region"} 130.0
# HELP openstack_neutron_network_ip_availabilities_total network_ip_availabilities_total
# TYPE openstack_neutron_network_ip_availabilities_total gauge
openstack_neutron_network_ip_availabilities_total{region="Region",network_id="00bd4d2d-e8d7-4715-a52d-f9c8378a8ab4",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="4bc6a4b06c11495c8beed2fecb3da5f7"} 253.0
openstack_neutron_network_ip_availabilities_total{region="Region",network_id="00de2fca-b8e4-42b8-84fa-1d88648e08eb",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="7abf4adfd30548a381554b3a4a08cd5d"} 253.0
# HELP openstack_neutron_network_ip_availabilities_used network_ip_availabilities_used
# TYPE openstack_neutron_network_ip_availabilities_used gauge
openstack_neutron_network_ip_availabilities_used{region="Region",network_id="00bd4d2d-e8d7-4715-a52d-f9c8378a8ab4",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="4bc6a4b06c11495c8beed2fecb3da5f7"} 4.0
openstack_neutron_network_ip_availabilities_used{region="Region",network_id="00de2fca-b8e4-42b8-84fa-1d88648e08eb",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="7abf4adfd30548a381554b3a4a08cd5d"} 5.0
# HELP openstack_neutron_security_groups security_groups
# TYPE openstack_neutron_security_groups gauge
# HELP openstack_neutron_port port
# TYPE openstack_neutron_port gauge
openstack_neutron_port{admin_state_up="true",binding_vif_type="",device_owner="network:router_gateway",mac_address="fa:16:3e:58:42:ed",network_id="70c1db1f-b701-45bd-96e0-a313ee3430b3",uuid="d80b1a3b-4fc1-49f3-952e-1e2ab7081d8b"} 1
openstack_neutron_port{admin_state_up="true",binding_vif_type="",device_owner="network:router_interface",mac_address="fa:16:3e:bb:3c:e4",network_id="f27aa545-cbdd-4907-b0c6-c9e8b039dcc2",uuid="f71a6703-d6de-4be1-a91a-a570ede1d159"} 1
openstack_neutron_port{admin_state_up="true",binding_vif_type="ovs",device_owner="neutron:LOADBALANCERV2",mac_address="fa:16:3e:0b:14:fd",network_id="675c54a5-a9f3-4f5e-a0b4-e026b29c217uuid="f0b24508-eb48-4530-a38b-c042df147101"} 1
# HELP openstack_neutron_ports{region="Region"} ports
# TYPE openstack_neutron_ports{region="Region"} gauge
openstack_neutron_ports 1063.0
# HELP openstack_neutron_router router
# TYPE openstack_neutron_router gauge
openstack_neutron_router{admin_state_up="true",external_network_id="78620e54-9ec2-4372-8b07-3ac2d02e0288",id="9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f",name="router2",project_id="a2a651cc26974de98c9a1f9aa88eb2e6",status="N/A"} 1
openstack_neutron_router{admin_state_up="true",external_network_id="78620e54-9ec2-4372-8b07-3ac2d02e0288",id="f8a44de0-fc8e-45df-93c7-f79bf3b01c95",name="router1",project_id="a2a651cc26974de98c9a1f9aa88eb2e6",status="ACTIVE"} 1
# HELP openstack_neutron_routers{region="Region"} routers
# TYPE openstack_neutron_routers{region="Region"} gauge
openstack_neutron_routers 134.0
openstack_neutron_security_groups{region="Region"} 114.0
# HELP openstack_neutron_subnet subnet
# TYPE openstack_neutron_subnet gauge
openstack_neutron_subnet{cidr="10.0.0.0/24",dns_nameservers="",enable_dhcp="true",gateway_ip="10.0.0.1",id="08eae331-0402-425a-923c-34f7cfe39c1b",name="private-subnet",network_id="db193ab3-96e3-4cb3-8fc5-05f4296d0324",tags="tag1,tag2",tenant_id="26a7980765d0414dbc1fc1f88cdb7e6e"} 1
openstack_neutron_subnet{cidr="10.10.0.0/24",dns_nameservers="",enable_dhcp="true",gateway_ip="10.10.0.1",id="12769bb8-6c3c-11ec-8124-002b67875abf",name="pooled-subnet-ipv4",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 1
openstack_neutron_subnet{cidr="192.0.0.0/8",dns_nameservers="",enable_dhcp="true",gateway_ip="192.0.0.1",id="54d6f61d-db07-451c-9ab3-b9609b6b6f0b",name="my_subnet",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 1
openstack_neutron_subnet{cidr="2001:db8::/64",dns_nameservers="",enable_dhcp="true",gateway_ip="2001:db8::1",id="f73defec-6c43-11ec-a08b-002b67875abf",name="pooled-subnet-ipv6",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 1
# HELP openstack_neutron_subnets subnets
# TYPE openstack_neutron_subnets gauge
openstack_neutron_subnets{region="Region"} 130.0
# HELP openstack_neutron_subnets_free subnets_free
# TYPE openstack_neutron_subnets_free gauge
openstack_neutron_subnets_free{ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 7
openstack_neutron_subnets_free{ip_version="4",prefix="10.10.0.0/21",prefix_length="25",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 14
openstack_neutron_subnets_free{ip_version="4",prefix="10.10.0.0/21",prefix_length="26",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 28
# HELP openstack_neutron_subnets_total subnets_total
# TYPE openstack_neutron_subnets_total gauge
openstack_neutron_subnets_total{ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 8
openstack_neutron_subnets_total{ip_version="4",prefix="10.10.0.0/21",prefix_length="25",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 16
openstack_neutron_subnets_total{ip_version="4",prefix="10.10.0.0/21",prefix_length="26",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 32
# HELP openstack_neutron_subnets_used subnets_used
# TYPE openstack_neutron_subnets_used gauge
openstack_neutron_subnets_used{ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 1
openstack_neutron_subnets_used{ip_version="4",prefix="10.10.0.0/21",prefix_length="25",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 0
openstack_neutron_subnets_used{ip_version="4",prefix="10.10.0.0/21",prefix_length="26",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 0
# HELP openstack_loadbalancer_amphora_status amphora_status
# TYPE openstack_loadbalancer_amphora_status gauge
openstack_loadbalancer_amphora_status{cert_expiration="2020-08-08T23:44:31Z",compute_id="667bb225-69aa-44b1-8908-694dc624c267",ha_ip="10.0.0.6",id="45f40289-0551-483a-b089-47214bc2a8a4",lb_network_ip="192.168.0.6",loadbalancer_id="882f2a9d-9d53-4bd0-b0e9-08e9d0de11f9",role="MASTER",status="READY"} 2
openstack_loadbalancer_amphora_status{cert_expiration="2020-08-08T23:44:30Z",compute_id="9cd0f9a2-fe12-42fc-a7e3-5b6fbbe20395",ha_ip="10.0.0.6",id="7f890893-ced0-46ed-8697-33415d070e5a",lb_network_ip="192.168.0.17",loadbalancer_id="882f2a9d-9d53-4bd0-b0e9-08e9d0de11f9",role="BACKUP",status="READY"} 2
# HELP openstack_loadbalancer_loadbalancer_status loadbalancer_status
# TYPE openstack_loadbalancer_loadbalancer_status gauge
openstack_loadbalancer_loadbalancer_status{id="607226db-27ef-4d41-ae89-f2a800e9c2db",name="best_load_balancer",operating_status="ONLINE",project_id="e3cd678b11784734bc366148aa37580e",provider="octavia",provisioning_status="ACTIVE",vip_address="203.0.113.50"} 0
# HELP openstack_loadbalancer_total_amphorae total_amphorae
# TYPE openstack_loadbalancer_total_amphorae gauge
openstack_loadbalancer_total_amphorae 2
# HELP openstack_loadbalancer_total_loadbalancers total_loadbalancers
# TYPE openstack_loadbalancer_total_loadbalancers gauge
openstack_loadbalancer_total_loadbalancers 1
# HELP openstack_loadbalancer_up up
# TYPE openstack_loadbalancer_up gauge
openstack_loadbalancer_up 1
# HELP openstack_nova_agent_state agent_state
# TYPE openstack_nova_agent_state counter
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-01",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-01",region="Region",service="nova-conductor",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-01",region="Region",service="nova-consoleauth",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-01",region="Region",service="nova-scheduler",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-02",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-02",region="Region",service="nova-conductor",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-02",region="Region",service="nova-consoleauth",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-02",region="Region",service="nova-scheduler",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-03",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-03",region="Region",service="nova-conductor",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-03",region="Region",service="nova-consoleauth",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-03",region="Region",service="nova-scheduler",zone="internal"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-04",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-05",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-06",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-07",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-09",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-10",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-01",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-02",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-03",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-04",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-05",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-07",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-08",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-09",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-10",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-11",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-12",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-13",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-15",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-17",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-18",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-19",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-20",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-21",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-22",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-23",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-24",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-25",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-26",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-27",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-28",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-29",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-31",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-32",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-34",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-35",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-36",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-37",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-38",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-39",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-40",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-42",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-43",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-44",region="Region",service="nova-compute",zone="nova"} 1.0
openstack_nova_agent_state{adminState="enabled",hostname="compute-node-extra-45",region="Region",service="nova-compute",zone="nova"} 1.0
# HELP openstack_nova_availability_zones availability_zones
# TYPE openstack_nova_availability_zones gauge
openstack_nova_availability_zones{region="Region"} 1.0
# HELP openstack_nova_current_workload current_workload
# TYPE openstack_nova_current_workload gauge
openstack_nova_current_workload{aggregate="",hostname="compute-node-01",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-02",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-03",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-04",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-05",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-06",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-07",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-09",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-10",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-01",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-02",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-03",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-04",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-05",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-07",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-08",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-09",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-10",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-11",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-12",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-13",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-15",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-17",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-18",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-19",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-20",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-21",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-22",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-23",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-24",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-25",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-26",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-27",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-28",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-29",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-31",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-32",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-34",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-35",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-36",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-37",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-38",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-39",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-40",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-42",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-43",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-44",region="Region"} 0.0
openstack_nova_current_workload{aggregate="",hostname="compute-node-extra-45",region="Region"} 0.0
# HELP openstack_nova_flavor flavor
# TYPE openstack_nova_flavor gauge
openstack_nova_flavor{disk="0",id="1",is_public="true",name="m1.tiny",ram="512",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="2",is_public="true",name="m1.small",ram="2048",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="3",is_public="true",name="m1.medium",ram="4096",vcpus="2"} 1
openstack_nova_flavor{disk="0",id="4",is_public="true",name="m1.large",ram="8192",vcpus="4"} 1
openstack_nova_flavor{disk="0",id="5",is_public="true",name="m1.xlarge",ram="16384",vcpus="8"} 1
openstack_nova_flavor{disk="0",id="6",is_public="true",name="m1.tiny.specs",ram="512",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="7",is_public="true",name="m1.small.description",ram="2048",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="8",is_public="false",name="m1.tiny.private",ram="512",vcpus="1"} 1
# HELP openstack_nova_flavors flavors
# TYPE openstack_nova_flavors gauge
openstack_nova_flavors{region="Region"} 8
# TYPE openstack_nova_free_disk_bytes gauge
openstack_nova_free_disk_bytes{aggregates="",availability_zone="",hostname="host1"} 1.103806595072e+12
# HELP openstack_nova_local_storage_available_bytes local_storage_available_bytes
# TYPE openstack_nova_local_storage_available_bytes gauge
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-01",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-02",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-03",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-04",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-05",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-06",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-07",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-09",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-10",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-01",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-02",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-03",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-04",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-05",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-07",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-08",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-09",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-10",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-11",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-12",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-13",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-15",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-17",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-18",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-19",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-20",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-21",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-22",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-23",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-24",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-25",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-26",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-27",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-28",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-29",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-31",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-32",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-34",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-35",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-36",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-37",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-38",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-39",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-40",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-42",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-43",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-44",region="Region"} 1.07823006482432e+14
openstack_nova_local_storage_available_bytes{aggregate="",hostname="compute-node-extra-45",region="Region"} 1.07823006482432e+14
# HELP openstack_nova_local_storage_used_bytes local_storage_used_bytes
# TYPE openstack_nova_local_storage_used_bytes gauge
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-01",region="Region"} 2.147483648e+11
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-02",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-03",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-04",region="Region"} 1.24554051584e+12
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-05",region="Region"} 1.7179869184e+11
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-06",region="Region"} 1.073741824e+12
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-07",region="Region"} 1.073741824e+12
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-09",region="Region"} 7.516192768e+11
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-10",region="Region"} 6.39950127104e+11
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-01",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-02",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-03",region="Region"} 4.422742573056e+12
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-04",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-05",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-07",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-08",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-09",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-10",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-11",region="Region"} 1.7179869184e+11
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-12",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-13",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-15",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-17",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-18",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-19",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-20",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-21",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-22",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-23",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-24",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-25",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-26",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-27",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-28",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-29",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-31",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-32",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-34",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-35",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-36",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-37",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-38",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-39",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-40",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-42",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-43",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-44",region="Region"} 0.0
openstack_nova_local_storage_used_bytes{aggregate="",hostname="compute-node-extra-45",region="Region"} 0.0
# HELP openstack_nova_memory_available_bytes memory_available_bytes
# TYPE openstack_nova_memory_available_bytes gauge
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-01",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-02",region="Region"} 6.751256576e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-03",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-04",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-05",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-06",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-07",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-09",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-10",region="Region"} 6.7513614336e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-01",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-02",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-03",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-04",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-05",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-07",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-08",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-09",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-10",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-11",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-12",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-13",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-15",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-17",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-18",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-19",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-20",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-21",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-22",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-23",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-24",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-25",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-26",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-27",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-28",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-29",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-31",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-32",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-34",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-35",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-36",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-37",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-38",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-39",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-40",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-42",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-43",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-44",region="Region"} 6.7542974464e+10
openstack_nova_memory_available_bytes{aggregate="",hostname="compute-node-extra-45",region="Region"} 6.7542974464e+10
# HELP openstack_nova_memory_used_bytes memory_used_bytes
# TYPE openstack_nova_memory_used_bytes gauge
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-01",region="Region"} 9.135194112e+09
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-02",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-03",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-04",region="Region"} 7.2049754112e+10
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-05",region="Region"} 9.135194112e+09
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-06",region="Region"} 2.5702694912e+10
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-07",region="Region"} 4.9308237824e+10
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-09",region="Region"} 1.3220446208e+10
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-10",region="Region"} 3.221225472e+10
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-01",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-02",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-03",region="Region"} 2.565865472e+09
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-04",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-05",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-07",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-08",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-09",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-10",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-11",region="Region"} 9.126805504e+09
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-12",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-13",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-15",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-17",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-18",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-19",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-20",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-21",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-22",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-23",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-24",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-25",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-26",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-27",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-28",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-29",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-31",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-32",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-34",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-35",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-36",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-37",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-38",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-39",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-40",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-42",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-43",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-44",region="Region"} 5.36870912e+08
openstack_nova_memory_used_bytes{aggregate="",hostname="compute-node-extra-45",region="Region"} 5.36870912e+08
# HELP openstack_nova_running_vms running_vms
# TYPE openstack_nova_running_vms gauge
openstack_nova_running_vms{aggregate="",hostname="compute-node-01",region="Region"} 1.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-02",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-03",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-04",region="Region"} 3.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-05",region="Region"} 1.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-06",region="Region"} 3.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-07",region="Region"} 4.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-09",region="Region"} 2.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-10",region="Region"} 6.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-01",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-02",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-03",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-04",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-05",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-07",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-08",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-09",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-10",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-11",region="Region"} 1.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-12",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-13",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-15",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-17",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-18",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-19",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-20",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-21",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-22",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-23",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-24",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-25",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-26",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-27",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-28",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-29",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-31",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-32",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-34",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-35",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-36",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-37",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-38",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-39",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-40",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-42",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-43",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-44",region="Region"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-node-extra-45",region="Region"} 0.0
# HELP openstack_nova_security_groups security_groups
# TYPE openstack_nova_security_groups gauge
openstack_nova_security_groups{region="Region"} 5.0
# HELP openstack_nova_server_local_gb server_local_gb
# TYPE openstack_nova_server_local_gb gauge
openstack_nova_server_local_gb{id="27bb2854-b06a-48f5-ab4e-139817b8b8ff",name="openstack-monitoring-0",tenant_id="110f6313d2d346b4aa90eabe4970b62a"} 10
# HELP openstack_nova_server_status server_status
# TYPE openstack_nova_server_status gauge
openstack_nova_server_status{address_ipv4="1.2.3.4",address_ipv6="80fe::",availability_zone="nova",flavor_id="1",host_id="2091634baaccdc4c5a1d57069c833e402921df696b7f970791b12ec6",hypervisor_hostname="fake-mini",id="2ce4c5b3-2866-4972-93ce-77a2ea46a7f9",name="new-server-test",status="ACTIVE",tenant_id="6f70656e737461636b20342065766572",user_id="fake",uuid="2ce4c5b3-2866-4972-93ce-77a2ea46a7f9"} 0
# HELP openstack_nova_total_vms total_vms
# TYPE openstack_nova_total_vms gauge
openstack_nova_total_vms{region="Region"} 23.0
# HELP openstack_nova_vcpus_available vcpus_available
# TYPE openstack_nova_vcpus_available gauge
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-01",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-02",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-03",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-04",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-05",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-06",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-07",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-09",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-10",region="Region"} 48.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-01",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-02",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-03",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-04",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-05",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-07",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-08",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-09",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-10",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-11",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-12",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-13",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-15",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-17",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-18",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-19",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-20",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-21",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-22",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-23",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-24",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-25",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-26",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-27",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-28",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-29",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-31",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-32",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-34",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-35",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-36",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-37",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-38",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-39",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-40",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-42",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-43",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-44",region="Region"} 8.0
openstack_nova_vcpus_available{aggregate="",hostname="compute-node-extra-45",region="Region"} 8.0
# HELP openstack_nova_vcpus_used vcpus_used
# TYPE openstack_nova_vcpus_used gauge
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-01",region="Region"} 8.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-02",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-03",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-04",region="Region"} 56.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-05",region="Region"} 8.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-06",region="Region"} 24.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-07",region="Region"} 41.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-09",region="Region"} 12.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-10",region="Region"} 25.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-01",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-02",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-03",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-04",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-05",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-07",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-08",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-09",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-10",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-11",region="Region"} 8.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-12",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-13",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-15",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-17",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-18",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-19",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-20",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-21",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-22",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-23",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-24",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-25",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-26",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-27",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-28",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-29",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-31",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-32",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-34",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-35",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-36",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-37",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-38",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-39",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-40",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-42",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-43",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-44",region="Region"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-node-extra-45",region="Region"} 0.0
# HELP openstack_object_store_objects objects
# TYPE openstack_object_store_objects gauge
openstack_object_store_objects{container_name="test2"} 1
# HELP openstack_object_store_up up
# TYPE openstack_object_store_up gauge
openstack_object_store_up 1
# HELP openstack_trove_instance_status instance_status
# TYPE openstack_trove_instance_status gauge
openstack_trove_instance_status{datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 2
# HELP openstack_trove_instance_volume_size_gb instance_volume_size_gb
# TYPE openstack_trove_instance_volume_size_gb gauge
openstack_trove_instance_volume_size_gb{datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 20
# HELP openstack_trove_instance_volume_used_gb instance_volume_used_gb
# TYPE openstack_trove_instance_volume_used_gb gauge
openstack_trove_instance_volume_used_gb{datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0.4
# HELP openstack_trove_total_instances total_instances
# TYPE openstack_trove_total_instances gauge
openstack_trove_total_instances 1
# HELP openstack_trove_up up
# TYPE openstack_trove_up gauge
openstack_trove_up 1
# HELP openstack_heat_stack_status stack_status
# TYPE openstack_heat_stack_status gauge
openstack_heat_stack_status{id="0009e826-5ad0-4310-994c-d3d2151eb6fd",name="demo-stack1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="UPDATE_COMPLETE"} 11
openstack_heat_stack_status{id="00cb0780-c883-4964-89c3-b79d840b3cbf",name="demo-stack2",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="CREATE_COMPLETE"} 5
openstack_heat_stack_status{id="03438d56-3109-4881-b75e-c8eb83cb9985",name="demo-stack3",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="CREATE_FAILED"} 4
openstack_heat_stack_status{id="1128f6cf-589b-468c-8ba1-9ae7e3f24507",name="demo-stack4",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="UPDATE_FAILED"} 10
openstack_heat_stack_status{id="23f50926-d2ab-4e13-86ee-0c768f8ce426",name="demo-stack5",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="DELETE_IN_PROGRESS"} 6
openstack_heat_stack_status{id="24cb54d6-f060-41b6-b7ae-e4c149b35382",name="demo-stack6",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="DELETE_FAILED"} 7
# HELP openstack_heat_stack_status_counter stack_status_counter
# TYPE openstack_heat_stack_status_counter gauge
openstack_heat_stack_status_counter{status="ADOPT_COMPLETE"} 0
openstack_heat_stack_status_counter{status="ADOPT_FAILED"} 0
openstack_heat_stack_status_counter{status="ADOPT_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="CHECK_COMPLETE"} 0
openstack_heat_stack_status_counter{status="CHECK_FAILED"} 0
openstack_heat_stack_status_counter{status="CHECK_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="CREATE_COMPLETE"} 1
openstack_heat_stack_status_counter{status="CREATE_FAILED"} 1
openstack_heat_stack_status_counter{status="CREATE_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="DELETE_COMPLETE"} 0
openstack_heat_stack_status_counter{status="DELETE_FAILED"} 1
openstack_heat_stack_status_counter{status="DELETE_IN_PROGRESS"} 1
openstack_heat_stack_status_counter{status="INIT_COMPLETE"} 0
openstack_heat_stack_status_counter{status="INIT_FAILED"} 0
openstack_heat_stack_status_counter{status="INIT_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="RESUME_COMPLETE"} 0
openstack_heat_stack_status_counter{status="RESUME_FAILED"} 0
openstack_heat_stack_status_counter{status="RESUME_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="ROLLBACK_COMPLETE"} 0
openstack_heat_stack_status_counter{status="ROLLBACK_FAILED"} 0
openstack_heat_stack_status_counter{status="ROLLBACK_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="SNAPSHOT_COMPLETE"} 0
openstack_heat_stack_status_counter{status="SNAPSHOT_FAILED"} 0
openstack_heat_stack_status_counter{status="SNAPSHOT_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="SUSPEND_COMPLETE"} 0
openstack_heat_stack_status_counter{status="SUSPEND_FAILED"} 0
openstack_heat_stack_status_counter{status="SUSPEND_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="UPDATE_COMPLETE"} 1
openstack_heat_stack_status_counter{status="UPDATE_FAILED"} 1
openstack_heat_stack_status_counter{status="UPDATE_IN_PROGRESS"} 0
# HELP openstack_heat_up up
# TYPE openstack_heat_up gauge
openstack_heat_up 1
# HELP openstack_placement_resource_allocation_ratio resource_allocation_ratio
# TYPE openstack_placement_resource_allocation_ratio gauge
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 1.299999952316284
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 3
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 1
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 1
# HELP openstack_placement_resource_reserved resource_reserved
# TYPE openstack_placement_resource_reserved gauge
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 0
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 8192
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 8192
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 0
# HELP openstack_placement_resource_total resource_total
# TYPE openstack_placement_resource_total gauge
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 2047
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 772447
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 96
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 2047
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 772447
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 96
# HELP openstack_placement_resource_usage resource_usage
# TYPE openstack_placement_resource_usage gauge
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 6969
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 1945
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 10
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 0
# HELP openstack_placement_up up
# TYPE openstack_placement_up gauge
openstack_placement_up 1
# HELP openstack_sharev2_share_gb share_gb
# TYPE openstack_sharev2_share_gb gauge
openstack_sharev2_share_gb{availability_zone="az1",id="4be93e2e-ffff-4362-ffff-603e3ec2a5d6",name="share-test",share_proto="NFS",share_type="az1",share_type_name="",status="available"} 1
# HELP openstack_sharev2_share_status share_status
# TYPE openstack_sharev2_share_status gauge
openstack_sharev2_share_status{id="4be93e2e-ffff-4362-ffff-603e3ec2a5d6",name="share-test",share_proto="NFS",share_type="az1",share_type_name="",size="1",status="available"} 1
# HELP openstack_sharev2_share_status_counter share_status_counter
# TYPE openstack_sharev2_share_status_counter gauge
openstack_sharev2_share_status_counter{status="available"} 1
openstack_sharev2_share_status_counter{status="creating"} 0
openstack_sharev2_share_status_counter{status="deleting"} 0
openstack_sharev2_share_status_counter{status="error"} 0
openstack_sharev2_share_status_counter{status="error_deleting"} 0
openstack_sharev2_share_status_counter{status="extending"} 0
openstack_sharev2_share_status_counter{status="inactive"} 0
openstack_sharev2_share_status_counter{status="managing"} 0
openstack_sharev2_share_status_counter{status="migrating"} 0
openstack_sharev2_share_status_counter{status="migration_error"} 0
openstack_sharev2_share_status_counter{status="restoring"} 0
openstack_sharev2_share_status_counter{status="reverting"} 0
openstack_sharev2_share_status_counter{status="reverting_error"} 0
openstack_sharev2_share_status_counter{status="reverting_to_snapshot"} 0
openstack_sharev2_share_status_counter{status="shrinking"} 0
openstack_sharev2_share_status_counter{status="shrinking_error"} 0
openstack_sharev2_share_status_counter{status="soft_deleting"} 0
openstack_sharev2_share_status_counter{status="unmanaging"} 0
openstack_sharev2_share_status_counter{status="updating"} 0
# HELP openstack_sharev2_shares_counter shares_counter
# TYPE openstack_sharev2_shares_counter gauge
openstack_sharev2_shares_counter 1
# HELP openstack_sharev2_up up
# TYPE openstack_sharev2_up gauge
openstack_sharev2_up 1

```

### Communication

Please join us at #openstack-exporter at [OFTC](https://www.oftc.net/)
