# OpenStack Exporter for Prometheus [![Build Status][buildstatus]][circleci] [![Docker Repository on Quay](https://quay.io/repository/niedbalski/openstack-exporter/status "Docker Repository on Quay")](https://quay.io/repository/niedbalski/openstack-exporter)

A [OpenStack](https://openstack.org/) exporter for prometheus written in Golang.

## Description

The OpenStack exporter, exports Prometheus metrics from a running OpenStack cloud
for consumption by prometheus. The cloud credentials and identity configuration
should use the [os-client-config](https://docs.openstack.org/os-client-config/latest/) format
and must by specified with the `--os-client-config` flag.

Other options as the binding address/port can by explored with the --help flag.

By default the openstack\_exporter serves on port `0.0.0.0:9180` at `/metrics`

```sh
make
./openstack-exporter --os-client-config /etc/openstack/clouds.yml region.mycludprovider.org
```

Alternatively a Dockerfile and image are supplied

```sh
docker run -p 9180:9180 quay.io/niedbalski/openstack-exporter:v0.0.6
```

### Command line options

The current list of command line options (by running --help)
```sh
usage: openstack-exporter [<flags>] <cloud>

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --web.listen-address=":9180"
                                 address:port to listen on
      --web.telemetry-path="/metrics"
                                 uri path to expose metrics
      --os-client-config="/etc/openstack/clouds.yml"
                                 Path to the cloud configuration file
      --prefix="openstack"       Prefix for metrics
      --disable-service.network  Disable the network service exporter
      --disable-service.compute  Disable the compute service exporter
      --disable-service.image    Disable the image service exporter
      --disable-service.volumev3   Disable the volumev3 service exporter
      --disable-service.identity
                                 Disable the identity service exporter

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
     verify: true | false
```

## Contributing

Please fill pull requests or issues under Github. Feel free to request any metrics
that might be missing.

## Metrics

The neutron/nova metrics contains the *_state metrics, which are separated
by service/agent name. 

Please note that by convention resources metrics such as memory or storage are returned in bytes.


Name     | Sample Labels | Sample Value | Description
---------|---------------|--------------|------------
openstack_neutron_agent_state|adminState="up",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"|1 or 0 (bool)
openstack_neutron_floating_ips|region="RegionOne"|4.0 (float)
openstack_neutron_networks|region="RegionOne"|25.0 (float)
openstack_neutron_subnets|region="RegionOne"|4.0 (float)
openstack_neutron_security_groups|region="RegionOne"|10.0 (float)
openstack_nova_availability_zones|region="RegionOne"|4.0 (float)
openstack_nova_flavors|region="RegionOne"|4.0 (float)
openstack_nova_total_vms|region="RegionOne"|12.0 (float)
openstack_nova_running_vms|region="RegionOne",hostname="compute-01"|12.0 (float)
openstack_nova_local_storage_used_bytes|region="RegionOne",hostname="compute-01"|100.0 (float)
openstack_nova_local_storage_available_bytes|region="RegionOne",hostname="compute-01"|30.0 (float)
openstack_nova_memory_used_bytes|region="RegionOne",hostname="compute-01"|40000.0 (float)
openstack_nova_memory_available_bytes|region="RegionOne",hostname="compute-01"|40000.0 (float)
openstack_nova_agent_state|hostname="compute-01",region="RegionOne",service="nova-compute",adminState="enabled",zone="nova"|1.0 or 0 (bool)
openstack_nova_vcpus_available|region="RegionOne",hostname="compute-01"|128.0 (float)
openstack_nova_vcpus_used|region="RegionOne",hostname="compute-01"|32.0 (float)
openstack_cinder_service_state|hostname="compute-01",region="RegionOne",service="cinder-backup",adminState="enabled",zone="nova"|1.0 or 0 (bool)
openstack_cinder_volumes|region="RegionOne"|4.0 (float)
openstack_cinder_snapshots|region="RegionOne"|4.0 (float)
openstack_identity_domains|region="RegionOne"|1.0 (float)
openstack_identity_users|region="RegionOne"|30.0 (float)
openstack_identity_projects|region="RegionOne"|33.0 (float)
openstack_identity_groups|region="RegionOne"|1.0 (float)
openstack_identity_regions|region="RegionOne"|1.0 (float)

## Example metrics
```
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
# HELP openstack_cinder_snapshots snapshots
# TYPE openstack_cinder_snapshots gauge
openstack_cinder_snapshots{region="Region"} 0.0
# HELP openstack_cinder_volumes volumes
# TYPE openstack_cinder_volumes gauge
openstack_cinder_volumes{region="Region"} 8.0
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
# HELP openstack_neutron_security_groups security_groups
# TYPE openstack_neutron_security_groups gauge
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
```

[buildstatus]: https://circleci.com/gh/openstack-exporter/openstack-exporter/tree/master.svg?style=shield
[circleci]: https://circleci.com/gh/openstack-exporter/openstack-exporter
[hub]: https://hub.docker.com/r/niedbalski/openstack-exporter/
