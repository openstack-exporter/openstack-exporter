# OpenStack Exporter for Prometheus [![Build Status][buildstatus]][circleci] 

A [OpenStack](https://openstack.org/) exporter for prometheus written in Golang using the
[gophercloud](https://github.com/gophercloud/gophercloud) library.

### Deployment options

The openstack-exporter can be deployed using the following mechanisms:

* Via docker images directly from our repositories
* Via snaps
* By using [kolla-ansible](https://github.com/opentack/kolla-ansible) by setting enable_prometheus_openstack_exporter: true
* By using [helm charts](https://github.com/openstack-exporter/helm-charts)

### Containers and binaries build status

amd64: [![Docker amd64 repository](https://quay.io/repository/niedbalski/openstack-exporter-linux-amd64/status "Docker amd64 Repository on Quay")](https://quay.io/repository/niedbalski/openstack-exporter-linux-amd64) | arm64: [![Docker amd64 repository](https://quay.io/repository/niedbalski/openstack-exporter-linux-arm64/status "Docker arm64 Repository on Quay")](https://quay.io/repository/niedbalski/openstack-exporter-linux-arm64)

### Latest Docker master images

```sh
docker pull quay.io/niedbalski/openstack-exporter-linux-amd64:master
docker pull quay.io/niedbalski/openstack-exporter-linux-arm64:master
```
### Latest Docker release images
```sh
docker pull quay.io/niedbalski/openstack-exporter-linux-amd64:v1.0.0
docker pull quay.io/niedbalski/openstack-exporter-linux-arm64:v1.0.0

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
and must by specified with the `--os-client-config` flag.

Other options as the binding address/port can by explored with the --help flag.

By default the openstack\_exporter serves on port `0.0.0.0:9180` at the `/metrics` URL.

You can build it by yourself by cloning this repository and run:

```sh
make common-build
./openstack-exporter --os-client-config /etc/openstack/clouds.yaml region.mycludprovider.org
```

Or alternatively you can use the docker images, as follows (check the openstack configuration section for configuration
details):

```sh
docker run -v "$HOME/.config/openstack/clouds.yml":/etc/openstack/clouds.yaml -it quay.io/niedbalski/openstack-exporter-linux-amd64:master my-cloud.org
```

### Command line options

The current list of command line options (by running --help)
```sh
usage: openstack-exporter [<flags>] <cloud>

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --log.level="info"         Log level: [debug, info, warn, error, fatal]
      --web.listen-address=":9180"  
                                 address:port to listen on
      --web.telemetry-path="/metrics"  
                                 uri path to expose metrics
      --os-client-config="/etc/openstack/clouds.yaml"  
                                 Path to the cloud configuration file
      --prefix="openstack"       Prefix for metrics
      --endpoint-type="public"   openstack endpoint type to use (i.e: public, internal, admin)
      --collect-metric-time      time spent collecting each metric
  -d, --disable-metric= ...      multiple --disable-metric can be specified in the format: service-metric (i.e: cinder-snapshots)
      --disable-service.network  Disable the network service exporter
      --disable-service.compute  Disable the compute service exporter
      --disable-service.image    Disable the image service exporter
      --disable-service.volume   Disable the volume service exporter
      --disable-service.identity  
                                 Disable the identity service exporter
      --disable-service.object-store
                                 Disable the object-store service exporter
      --disable-service.load-balancer
                                 Disable the load-balancer service exporter
      --disable-service.container-infra
                                 Disable the container-infra service exporter
      --disable-service.dns      Disable the dns service exporter
      --version                  Show application version.

Args:
  <cloud>  name or id of the cloud to gather metrics from
```

### OpenStack configuration

The cloud credentials and identity configuration
should use the [os-client-config](https://docs.openstack.org/os-client-config/latest/) format
and must by specified with the `--os-client-config` flag.

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
     user_domain_name: 'Default'
     auth_url: {{ admin_protocol }}://{{ kolla_internal_fqdn }}:{{ keystone_admin_port }}/v3
     cacert: |
            ---- BEGIN CERTIFICATE ---
      ...
    verify: true | false  // disable || enable SSL certificate verification
```

## Contributing

Please fill pull requests or issues under Github. Feel free to request any metrics
that might be missing.

### Communication

Please join us at #openstack-exporter at Freenode

## Metrics

The neutron/nova metrics contains the *_state metrics, which are separated
by service/agent name.

Please note that by convention resources metrics such as memory or storage are returned in bytes.


Name     | Sample Labels | Sample Value | Description
---------|---------------|--------------|------------
openstack_neutron_agent_state|adminState="up",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"|1 or 0 (bool)
openstack_neutron_floating_ips|region="RegionOne"|4.0 (float)
openstack_neutron_networks|region="RegionOne"|25.0 (float)
openstack_neutron_ports|region="RegionOne"| 1063.0 (float)
openstack_neutron_subnets|region="RegionOne"|4.0 (float)
openstack_neutron_security_groups|region="RegionOne"|10.0 (float)
openstack_neutron_network_ip_availabilities_total|region="RegionOne",network_id="23046ac4-67fc-4bf6-842b-875880019947",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="478340c7c6bf49c99ce40641fd13ba96"|253.0 (float)
openstack_neutron_network_ip_availabilities_used|region="RegionOne",network_id="23046ac4-67fc-4bf6-842b-875880019947",network_name="default-network",cidr="10.0.0.0/16",subnet_name="my-subnet",project_id="478340c7c6bf49c99ce40641fd13ba96"|151.0 (float)
openstack_neutron_routers|region="RegionOne"|134.0 (float)
openstack_nova_availability_zones|region="RegionOne"|4.0 (float)
openstack_nova_flavors|region="RegionOne"|4.0 (float)
openstack_nova_total_vms|region="RegionOne"|12.0 (float)
openstack_nova_server_status|region="RegionOne",hostname="compute-01""id", "name", "tenant_id", "user_id", "address_ipv4",                                                                     	"address_ipv6", "host_id", "uuid", "availability_zone"|0.0 (float)
openstack_nova_running_vms|region="RegionOne",hostname="compute-01",availability_zone="az1",aggregates="shared,ssd"|12.0 (float)
openstack_nova_local_storage_used_bytes|region="RegionOne",hostname="compute-01",aggregates="shared,ssd"|100.0 (float)
openstack_nova_local_storage_available_bytes|region="RegionOne",hostname="compute-01",aggregates="shared,ssd"|30.0 (float)
openstack_nova_memory_used_bytes|region="RegionOne",hostname="compute-01",aggregates="shared,ssd"|40000.0 (float)
openstack_nova_memory_available_bytes|region="RegionOne",hostname="compute-01",aggregates="shared,ssd"|40000.0 (float)
openstack_nova_agent_state|hostname="compute-01",region="RegionOne", id="288", service="nova-compute",adminState="enabled",zone="nova"|1.0 or 0 (bool)
openstack_nova_vcpus_available|region="RegionOne",hostname="compute-01",aggregates="shared,ssd"|128.0 (float)
openstack_nova_vcpus_used|region="RegionOne",hostname="compute-01",aggregates="shared,ssd"|32.0 (float)
openstack_nova_limits_vcpus_max|tenant="demo-project"|128.0 (float)
openstack_nova_limits_vcpus_used|tenant="demo-project"|32.0 (float)
openstack_nova_limits_memory_max|tenant="demo-project"|40000.0 (float)
openstack_nova_limits_memory_used|tenant="demo-project"|40000.0 (float)
openstack_cinder_service_state|hostname="compute-01",region="RegionOne",service="cinder-backup",adminState="enabled",zone="nova"|1.0 or 0 (bool)
openstack_cinder_limits_volume_max_gb|tenant="demo-project",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"|40000.0 (float)
openstack_cinder_limits_volume_used_gb|tenant="demo-project",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"|40000.0 (float)
openstack_cinder_volumes|region="RegionOne"|4.0 (float)
openstack_cinder_snapshots|region="RegionOne"|4.0 (float)
openstack_cinder_volume_status|region="RegionOne""id", "name", "status", "bootable", "tenant_id", "size", "volume_type"|4.0 (float) 
openstack_designate_zones| region="RegionOne"|4.0 (float)
openstack_designate_zone_status| region="RegionOne""id", "name", "status", "tenant_id", "type"|4.0 (float)
openstack_designate_recordsets| region="RegionOne"|4.0 (float)
openstack_designate_recordsets_status| region="RegionOne""id", "name", "status", "zone_id", "zone_name", "type"|4.0 (float)
openstack_identity_domains|region="RegionOne"|1.0 (float)
openstack_identity_users|region="RegionOne"|30.0 (float)
openstack_identity_projects|region="RegionOne"|33.0 (float)
openstack_identity_groups|region="RegionOne"|1.0 (float)
openstack_identity_regions|region="RegionOne"|1.0 (float)
openstack_object_store_objects|region="RegionOne",container_name="test2"|1.0 (float) 
openstack_metric_collect_seconds | {openstack_metric="agent_state",openstack_service="openstack_cinder"} |1.27843913| Only if --collect-metric-time is passed

## Example metrics
```
# HELP openstack_cinder_agent_state agent_state
# TYPE openstack_cinder_agent_state counter
openstack_cinder_volume_status{bootable="",id="11017190-61ab-426f-9366-2299292sadssad",name="",region="Region",size="0",status="",tenant_id="",volume_type=""} 1.0
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
openstack_designate_recordsets 1
# HELP openstack_designate_recordsets_status recordsets_status
# TYPE openstack_designate_recordsets_status gauge
openstack_designate_recordsets_status{ZoneName="example.com.",id="f7b10e9b-0cae-4a91-b162-562bc6096648",name="example.org.",status="PENDING",type="A",zone_id="2150b1bf-dee2-4221-9d85-11f7886fb15f"} 0
# HELP openstack_designate_up up
# TYPE openstack_designate_up gauge
openstack_designate_up 1
# HELP openstack_designate_zone_status zone_status
# TYPE openstack_designate_zone_status gauge
openstack_designate_zone_status{id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",name="example.org.",status="ACTIVE",tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",type="PRIMARY"} 1
# HELP openstack_designate_zones zones
# TYPE openstack_designate_zones gauge
openstack_designate_zones 1
# HELP openstack_container_infra_cluster_status cluster_status
# TYPE openstack_container_infra_cluster_status gauge
openstack_container_infra_cluster_status{master_count="1",name="k8s",node_count="1",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_total_clusters total_clusters
# TYPE openstack_container_infra_total_clusters gauge
openstack_container_infra_total_clusters 1
# HELP openstack_glance_images images
# TYPE openstack_glance_images gauge
openstack_glance_images{region="Region"} 18.0
# HELP openstack_identity_domains domains
# TYPE openstack_identity_domains gauge
openstack_identity_domains{region="Region"} 1.0
# HELP openstack_identity_groups groups
# TYPE openstack_identity_groups gauge
openstack_identity_groups{region="Region"} 0.0
# HELP openstack_identity_projects projects
# TYPE openstack_identity_projects gauge
openstack_identity_projects{region="Region"} 33.0
# HELP openstack_identity_regions regions
# TYPE openstack_identity_regions gauge
openstack_identity_regions{region="Region"} 1.0
# HELP openstack_identity_users users
# TYPE openstack_identity_users gauge
openstack_identity_users{region="Region"} 39.0
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
# HELP openstack_neutron_floating_ips floating_ips
# TYPE openstack_neutron_floating_ips gauge
openstack_neutron_floating_ips{region="Region"} 22.0
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
# HELP openstack_neutron_ports{region="Region"} ports
# TYPE openstack_neutron_ports{region="Region"} gauge
openstack_neutron_ports 1063.0
# HELP openstack_neutron_routers{region="Region"} routers
# TYPE openstack_neutron_routers{region="Region"} gauge
openstack_neutron_routers 134.0
openstack_neutron_security_groups{region="Region"} 114.0
# HELP openstack_neutron_subnets subnets
# TYPE openstack_neutron_subnets gauge
openstack_neutron_subnets{region="Region"} 130.0
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
# HELP openstack_nova_flavors flavors
# TYPE openstack_nova_flavors gauge
openstack_nova_flavors{region="Region"} 6.0
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
```

[buildstatus]: https://circleci.com/gh/openstack-exporter/openstack-exporter/tree/master.svg?style=shield
[circleci]: https://circleci.com/gh/openstack-exporter/openstack-exporter
