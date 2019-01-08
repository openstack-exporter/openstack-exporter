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
docker run -p 9180:9180 quay.io/niedbalski/openstack-exporter:v0.0.3
```

## Contributing

Please fill pull requests or issues under Github. Feel free to request any metrics
that might be missing.

## Metrics

The neutron/nova metrics contains the *_state metrics, which are separated
by service/agent name. 

Name     | Sample Labels | Sample Value | Description
---------|---------------|--------------|------------
openstack_neutron_agent_state|adminState="up",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"|1 or 0 (bool)
openstack_neutron_floating_ips|region="RegionOne"|4.0 (float)
openstack_neutron_networks|region="RegionOne"|25.0 (float)
openstack_neutron_subnets|region="RegionOne"|4.0 (float)
openstack_neutron_security_groups|region="RegionOne"|10.0 (float)
openstack_nova_availability_zones|region="RegionOne"|4.0 (float)
openstack_nova_flavors|region="RegionOne"|4.0 (float)
openstack_nova_local_gb|region="RegionOne",hostname="compute-01"|100.0 (float)
openstack_nova_local_gb_used|region="RegionOne",hostname="compute-01"|30.0 (float)
openstack_nova_memory_mb|region="RegionOne",hostname="compute-01"|40000.0 (float)
openstack_nova_memory_mb_used|region="RegionOne",hostname="compute-01"|40000.0 (float)
openstack_nova_running_vms|region="RegionOne",hostname="compute-01"|12.0 (float)
openstack_nova_service_state|hostname="compute-01",region="RegionOne",service="nova-compute",status="enabled",zone="nova"|1.0 or 0 (bool)
openstack_nova_vcpus|region="RegionOne",hostname="compute-01"|128.0 (float)
openstack_nova_vcpus_used|region="RegionOne",hostname="compute-01"|32.0 (float)
openstack_cinder_service_state|hostname="compute-01",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"|1.0 or 0 (bool)
openstack_cinder_volumes|region="RegionOne"|4.0 (float)
openstack_cinder_snapshots|region="RegionOne"|4.0 (float)
openstack_glance_images|region="RegionOne"|4.0 (float)

## Example metrics
```
# HELP openstack_cinder_service_state service_state
# TYPE openstack_cinder_service_state counter
openstack_cinder_service_state{hostname="compute-01",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-01",region="RegionOne",service="cinder-scheduler",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-01@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-02",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-02",region="RegionOne",service="cinder-scheduler",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-02@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-03",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-03",region="RegionOne",service="cinder-scheduler",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-03@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-04",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-04@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
openstack_cinder_service_state{hostname="compute-05",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
# HELP openstack_cinder_snapshots snapshots
# TYPE openstack_cinder_snapshots gauge
openstack_cinder_snapshots{region="RegionOne"} 0.0
# HELP openstack_cinder_volumes volumes
# TYPE openstack_cinder_volumes gauge
openstack_cinder_volumes{region="RegionOne"} 8.0
# HELP openstack_glance_images images
# TYPE openstack_glance_images gauge
openstack_glance_images{region="RegionOne"} 18.0
# HELP openstack_neutron_agent_state agent_state
# TYPE openstack_neutron_agent_state counter
openstack_neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-l3-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-metadata-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-dhcp-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-l3-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-metadata-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-dhcp-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-l3-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-metadata-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-04",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-05",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-06",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-07",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-09",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-10",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
openstack_neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
# HELP openstack_neutron_floating_ips floating_ips
# TYPE openstack_neutron_floating_ips gauge
openstack_neutron_floating_ips{region="RegionOne"} 21.0
# HELP openstack_neutron_networks networks
# TYPE openstack_neutron_networks gauge
openstack_neutron_networks{region="RegionOne"} 128.0
# HELP openstack_neutron_security_groups security_groups
# TYPE openstack_neutron_security_groups gauge
openstack_neutron_security_groups{region="RegionOne"} 113.0
# HELP openstack_neutron_subnets subnets
# TYPE openstack_neutron_subnets gauge
openstack_neutron_subnets{region="RegionOne"} 128.0
# HELP openstack_nova_availability_zones availability_zones
# TYPE openstack_nova_availability_zones gauge
openstack_nova_availability_zones{region="RegionOne"} 1.0
# HELP openstack_nova_flavors flavors
# TYPE openstack_nova_flavors gauge
openstack_nova_flavors{region="RegionOne"} 6.0
# HELP openstack_nova_local_gb local_gb
# TYPE openstack_nova_local_gb gauge
openstack_nova_local_gb{aggregate="",hostname="compute-01",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-02",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-03",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-04",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-05",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-06",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-07",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-09",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-10",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-01",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-02",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-03",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-04",region="RegionOne"} 100418.0
openstack_nova_local_gb{aggregate="",hostname="compute-05",region="RegionOne"} 100418.0
# HELP openstack_nova_local_gb_used local_gb_used
# TYPE openstack_nova_local_gb_used gauge
openstack_nova_local_gb_used{aggregate="",hostname="compute-01",region="RegionOne"} 200.0
openstack_nova_local_gb_used{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
openstack_nova_local_gb_used{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
openstack_nova_local_gb_used{aggregate="",hostname="compute-04",region="RegionOne"} 1160.0
openstack_nova_local_gb_used{aggregate="",hostname="compute-05",region="RegionOne"} 160.0
# HELP openstack_nova_memory_mb memory_mb
# TYPE openstack_nova_memory_mb gauge
openstack_nova_memory_mb{aggregate="",hostname="compute-01",region="RegionOne"} 64386.0
openstack_nova_memory_mb{aggregate="",hostname="compute-02",region="RegionOne"} 64385.0
openstack_nova_memory_mb{aggregate="",hostname="compute-03",region="RegionOne"} 64386.0
openstack_nova_memory_mb{aggregate="",hostname="compute-04",region="RegionOne"} 64386.0
openstack_nova_memory_mb{aggregate="",hostname="compute-05",region="RegionOne"} 64386.0
openstack_nova_memory_mb{aggregate="",hostname="compute-06",region="RegionOne"} 64386.0
# HELP openstack_nova_memory_mb_used memory_mb_used
# TYPE openstack_nova_memory_mb_used gauge
openstack_nova_memory_mb_used{aggregate="",hostname="compute-01",region="RegionOne"} 8712.0
openstack_nova_memory_mb_used{aggregate="",hostname="compute-02",region="RegionOne"} 512.0
openstack_nova_memory_mb_used{aggregate="",hostname="compute-03",region="RegionOne"} 512.0
openstack_nova_memory_mb_used{aggregate="",hostname="compute-04",region="RegionOne"} 68712.0
openstack_nova_memory_mb_used{aggregate="",hostname="compute-05",region="RegionOne"} 8712.0
openstack_nova_memory_mb_used{aggregate="",hostname="compute-06",region="RegionOne"} 24512.0
# HELP openstack_nova_running_vms running_vms
# TYPE openstack_nova_running_vms gauge
openstack_nova_running_vms{aggregate="",hostname="compute-01",region="RegionOne"} 1.0
openstack_nova_running_vms{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
openstack_nova_running_vms{aggregate="",hostname="compute-04",region="RegionOne"} 3.0
openstack_nova_running_vms{aggregate="",hostname="compute-05",region="RegionOne"} 1.0
openstack_nova_running_vms{aggregate="",hostname="compute-06",region="RegionOne"} 3.0
# HELP openstack_nova_security_groups security_groups
# TYPE openstack_nova_security_groups gauge
openstack_nova_security_groups{region="RegionOne"} 5.0
# HELP openstack_nova_servers servers
# TYPE openstack_nova_servers gauge
openstack_nova_servers{region="RegionOne"} 22.0
# HELP openstack_nova_service_state service_state
# TYPE openstack_nova_service_state counter
openstack_nova_service_state{hostname="compute-01",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
openstack_nova_service_state{hostname="compute-01",region="RegionOne",service="nova-conductor",status="enabled",zone="internal"} 1.0
openstack_nova_service_state{hostname="compute-01",region="RegionOne",service="nova-consoleauth",status="enabled",zone="internal"} 1.0
openstack_nova_service_state{hostname="compute-01",region="RegionOne",service="nova-scheduler",status="enabled",zone="internal"} 1.0
openstack_nova_service_state{hostname="compute-02",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
openstack_nova_service_state{hostname="compute-02",region="RegionOne",service="nova-conductor",status="enabled",zone="internal"} 1.0
# HELP openstack_nova_vcpus vcpus
# TYPE openstack_nova_vcpus gauge
openstack_nova_vcpus{aggregate="",hostname="compute-01",region="RegionOne"} 48.0
openstack_nova_vcpus{aggregate="",hostname="compute-02",region="RegionOne"} 48.0
openstack_nova_vcpus{aggregate="",hostname="compute-03",region="RegionOne"} 48.0
openstack_nova_vcpus{aggregate="",hostname="compute-04",region="RegionOne"} 48.0
openstack_nova_vcpus{aggregate="",hostname="compute-05",region="RegionOne"} 48.0
openstack_nova_vcpus{aggregate="",hostname="compute-06",region="RegionOne"} 48.0
# HELP openstack_nova_vcpus_used vcpus_used
# TYPE openstack_nova_vcpus_used gauge
openstack_nova_vcpus_used{aggregate="",hostname="compute-01",region="RegionOne"} 8.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-04",region="RegionOne"} 56.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-05",region="RegionOne"} 8.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-06",region="RegionOne"} 24.0
openstack_nova_vcpus_used{aggregate="",hostname="compute-07",region="RegionOne"} 41.0
```

[buildstatus]: https://circleci.com/gh/niedbalski/openstack-exporter/tree/master.svg?style=shield
[circleci]: https://circleci.com/gh/niedbalski/openstack-exporter
[hub]: https://hub.docker.com/r/niedbalski/openstack-exporter/
