# OpenStack Exporter for Prometheus [![Build Status][buildstatus]][circleci] [![Docker Repository on Quay](https://quay.io/repository/niedbalski/openstack-exporter/status "Docker Repository on Quay")](https://quay.io/repository/niedbalski/openstack-exporter)

A [OpenStack](https://openstack.org/) exporter for prometheus written in Golang.

## Description

The openstack exporter exports metrics from a running OpenStack cloud
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
docker run -p 9180:9180 quay.io/niedbalski/openstack-exporter:v0.0.2
```

## Metrics

The neutron/nova metrics contains the *_state metrics, which are separated
by service/agent name. 

Name     | Sample Labels | Sample Value | Description
---------|---------------|--------------|------------
neutron_agent_state|adminState="up",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"|1 or 0 (bool)
neutron_floating_ips|region="RegionOne"|4.0 (float)
neutron_networks|region="RegionOne"|25.0 (float)
neutron_subnets|region="RegionOne"|4.0 (float)
neutron_security_groups|region="RegionOne"|10.0 (float)
nova_availability_zones|region="RegionOne"|4.0 (float)
nova_flavors|region="RegionOne"|4.0 (float)
nova_local_gb|region="RegionOne",hostname="compute-01"|100.0 (float)
nova_local_gb_used|region="RegionOne",hostname="compute-01"|30.0 (float)
nova_memory_mb|region="RegionOne",hostname="compute-01"|40000.0 (float)
nova_memory_mb_used|region="RegionOne",hostname="compute-01"|40000.0 (float)
nova_running_vms|region="RegionOne",hostname="compute-01"|12.0 (float)
nova_service_state|hostname="compute-01",region="RegionOne",service="nova-compute",status="enabled",zone="nova"|1.0 or 0 (bool)
nova_vcpus|region="RegionOne",hostname="compute-01"|128.0 (float)
nova_vcpus_used|region="RegionOne",hostname="compute-01"|32.0 (float)
cinder_service_state|hostname="compute-01",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"|1.0 or 0 (bool)
cinder_volumes|region="RegionOne"|4.0 (float)
cinder_snapshots|region="RegionOne"|4.0 (float)
glance_images|region="RegionOne"|4.0 (float)

## Example metrics
```
# HELP cinder_service_state service_state
# TYPE cinder_service_state counter
cinder_service_state{hostname="compute-01",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-01",region="RegionOne",service="cinder-scheduler",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-01@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-02",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-02",region="RegionOne",service="cinder-scheduler",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-02@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-03",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-03",region="RegionOne",service="cinder-scheduler",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-03@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-04",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-04@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-05",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-05@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-06",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-06@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-07",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-07@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-09",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-09@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-10",region="RegionOne",service="cinder-backup",status="enabled",zone="nova"} 1.0
cinder_service_state{hostname="compute-10@rbd-1",region="RegionOne",service="cinder-volume",status="enabled",zone="nova"} 1.0
# HELP cinder_snapshots snapshots
# TYPE cinder_snapshots gauge
cinder_snapshots{region="RegionOne"} 0.0
# HELP cinder_volumes volumes
# TYPE cinder_volumes gauge
cinder_volumes{region="RegionOne"} 8.0
# HELP glance_images images
# TYPE glance_images gauge
glance_images{region="RegionOne"} 18.0
# HELP neutron_agent_state agent_state
# TYPE neutron_agent_state counter
neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-dhcp-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-l3-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-metadata-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-dhcp-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-l3-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-metadata-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-dhcp-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-l3-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-metadata-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-04",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-05",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-06",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-07",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-09",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-10",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-01",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-02",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-03",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-04",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-05",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-07",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-08",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-09",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-10",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-11",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-12",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-13",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-15",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-17",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-18",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-19",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-20",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-21",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-22",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-23",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-24",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-25",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-26",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-27",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-28",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-29",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-31",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-32",region="RegionOne",service="neutron-openvswitch-agent"} 0.0
neutron_agent_state{adminState="up",hostname="compute-34",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-35",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-36",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-37",region="RegionOne",service="neutron-openvswitch-agent"} 0.0
neutron_agent_state{adminState="up",hostname="compute-38",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-39",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-40",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-42",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-43",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-44",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
neutron_agent_state{adminState="up",hostname="compute-45",region="RegionOne",service="neutron-openvswitch-agent"} 1.0
# HELP neutron_floating_ips floating_ips
# TYPE neutron_floating_ips gauge
neutron_floating_ips{region="RegionOne"} 21.0
# HELP neutron_networks networks
# TYPE neutron_networks gauge
neutron_networks{region="RegionOne"} 128.0
# HELP neutron_security_groups security_groups
# TYPE neutron_security_groups gauge
neutron_security_groups{region="RegionOne"} 113.0
# HELP neutron_subnets subnets
# TYPE neutron_subnets gauge
neutron_subnets{region="RegionOne"} 128.0
# HELP nova_availability_zones availability_zones
# TYPE nova_availability_zones gauge
nova_availability_zones{region="RegionOne"} 1.0
# HELP nova_flavors flavors
# TYPE nova_flavors gauge
nova_flavors{region="RegionOne"} 6.0
# HELP nova_local_gb local_gb
# TYPE nova_local_gb gauge
nova_local_gb{aggregate="",hostname="compute-01",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-02",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-03",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-04",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-05",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-06",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-07",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-09",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-10",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-01",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-02",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-03",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-04",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-05",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-07",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-08",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-09",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-10",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-11",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-12",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-13",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-15",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-17",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-18",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-19",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-20",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-21",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-22",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-23",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-24",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-25",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-26",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-27",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-28",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-29",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-31",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-32",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-34",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-35",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-36",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-37",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-38",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-39",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-40",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-42",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-43",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-44",region="RegionOne"} 100418.0
nova_local_gb{aggregate="",hostname="compute-45",region="RegionOne"} 100418.0
# HELP nova_local_gb_used local_gb_used
# TYPE nova_local_gb_used gauge
nova_local_gb_used{aggregate="",hostname="compute-01",region="RegionOne"} 200.0
nova_local_gb_used{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-04",region="RegionOne"} 1160.0
nova_local_gb_used{aggregate="",hostname="compute-05",region="RegionOne"} 160.0
nova_local_gb_used{aggregate="",hostname="compute-06",region="RegionOne"} 1000.0
nova_local_gb_used{aggregate="",hostname="compute-07",region="RegionOne"} 1000.0
nova_local_gb_used{aggregate="",hostname="compute-09",region="RegionOne"} 700.0
nova_local_gb_used{aggregate="",hostname="compute-10",region="RegionOne"} 396.0
nova_local_gb_used{aggregate="",hostname="compute-01",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-04",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-05",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-07",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-08",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-09",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-10",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-11",region="RegionOne"} 160.0
nova_local_gb_used{aggregate="",hostname="compute-12",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-13",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-15",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-17",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-18",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-19",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-20",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-21",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-22",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-23",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-24",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-25",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-26",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-27",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-28",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-29",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-31",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-32",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-34",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-35",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-36",region="RegionOne"} 3659.0
nova_local_gb_used{aggregate="",hostname="compute-37",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-38",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-39",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-40",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-42",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-43",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-44",region="RegionOne"} 0.0
nova_local_gb_used{aggregate="",hostname="compute-45",region="RegionOne"} 0.0
# HELP nova_memory_mb memory_mb
# TYPE nova_memory_mb gauge
nova_memory_mb{aggregate="",hostname="compute-01",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-02",region="RegionOne"} 64385.0
nova_memory_mb{aggregate="",hostname="compute-03",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-04",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-05",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-06",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-07",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-09",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-10",region="RegionOne"} 64386.0
nova_memory_mb{aggregate="",hostname="compute-01",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-02",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-03",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-04",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-05",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-07",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-08",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-09",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-10",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-11",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-12",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-13",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-15",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-17",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-18",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-19",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-20",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-21",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-22",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-23",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-24",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-25",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-26",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-27",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-28",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-29",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-31",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-32",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-34",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-35",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-36",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-37",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-38",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-39",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-40",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-42",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-43",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-44",region="RegionOne"} 64414.0
nova_memory_mb{aggregate="",hostname="compute-45",region="RegionOne"} 64414.0
# HELP nova_memory_mb_used memory_mb_used
# TYPE nova_memory_mb_used gauge
nova_memory_mb_used{aggregate="",hostname="compute-01",region="RegionOne"} 8712.0
nova_memory_mb_used{aggregate="",hostname="compute-02",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-03",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-04",region="RegionOne"} 68712.0
nova_memory_mb_used{aggregate="",hostname="compute-05",region="RegionOne"} 8712.0
nova_memory_mb_used{aggregate="",hostname="compute-06",region="RegionOne"} 24512.0
nova_memory_mb_used{aggregate="",hostname="compute-07",region="RegionOne"} 47024.0
nova_memory_mb_used{aggregate="",hostname="compute-09",region="RegionOne"} 12608.0
nova_memory_mb_used{aggregate="",hostname="compute-10",region="RegionOne"} 26624.0
nova_memory_mb_used{aggregate="",hostname="compute-01",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-02",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-03",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-04",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-05",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-07",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-08",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-09",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-10",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-11",region="RegionOne"} 8704.0
nova_memory_mb_used{aggregate="",hostname="compute-12",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-13",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-15",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-17",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-18",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-19",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-20",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-21",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-22",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-23",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-24",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-25",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-26",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-27",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-28",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-29",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-31",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-32",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-34",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-35",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-36",region="RegionOne"} 2126.0
nova_memory_mb_used{aggregate="",hostname="compute-37",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-38",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-39",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-40",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-42",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-43",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-44",region="RegionOne"} 512.0
nova_memory_mb_used{aggregate="",hostname="compute-45",region="RegionOne"} 512.0
# HELP nova_running_vms running_vms
# TYPE nova_running_vms gauge
nova_running_vms{aggregate="",hostname="compute-01",region="RegionOne"} 1.0
nova_running_vms{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-04",region="RegionOne"} 3.0
nova_running_vms{aggregate="",hostname="compute-05",region="RegionOne"} 1.0
nova_running_vms{aggregate="",hostname="compute-06",region="RegionOne"} 3.0
nova_running_vms{aggregate="",hostname="compute-07",region="RegionOne"} 4.0
nova_running_vms{aggregate="",hostname="compute-09",region="RegionOne"} 2.0
nova_running_vms{aggregate="",hostname="compute-10",region="RegionOne"} 5.0
nova_running_vms{aggregate="",hostname="compute-01",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-04",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-05",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-07",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-08",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-09",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-10",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-11",region="RegionOne"} 1.0
nova_running_vms{aggregate="",hostname="compute-12",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-13",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-15",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-17",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-18",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-19",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-20",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-21",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-22",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-23",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-24",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-25",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-26",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-27",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-28",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-29",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-31",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-32",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-34",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-35",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-36",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-37",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-38",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-39",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-40",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-42",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-43",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-44",region="RegionOne"} 0.0
nova_running_vms{aggregate="",hostname="compute-45",region="RegionOne"} 0.0
# HELP nova_security_groups security_groups
# TYPE nova_security_groups gauge
nova_security_groups{region="RegionOne"} 5.0
# HELP nova_servers servers
# TYPE nova_servers gauge
nova_servers{region="RegionOne"} 22.0
# HELP nova_service_state service_state
# TYPE nova_service_state counter
nova_service_state{hostname="compute-01",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-01",region="RegionOne",service="nova-conductor",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-01",region="RegionOne",service="nova-consoleauth",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-01",region="RegionOne",service="nova-scheduler",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-02",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-02",region="RegionOne",service="nova-conductor",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-02",region="RegionOne",service="nova-consoleauth",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-02",region="RegionOne",service="nova-scheduler",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-03",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-03",region="RegionOne",service="nova-conductor",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-03",region="RegionOne",service="nova-consoleauth",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-03",region="RegionOne",service="nova-scheduler",status="enabled",zone="internal"} 1.0
nova_service_state{hostname="compute-04",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-05",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-06",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-07",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-09",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-10",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-01",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-02",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-03",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-04",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-05",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-07",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-08",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-09",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-10",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-11",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-12",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-13",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-15",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-17",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-18",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-19",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-20",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-21",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-22",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-23",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-24",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-25",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-26",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-27",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-28",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-29",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-31",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-32",region="RegionOne",service="nova-compute",status="disabled",zone="nova"} 0.0
nova_service_state{hostname="compute-34",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-35",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-36",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-37",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 0.0
nova_service_state{hostname="compute-38",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-39",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-40",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-42",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-43",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-44",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
nova_service_state{hostname="compute-45",region="RegionOne",service="nova-compute",status="enabled",zone="nova"} 1.0
# HELP nova_vcpus vcpus
# TYPE nova_vcpus gauge
nova_vcpus{aggregate="",hostname="compute-01",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-02",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-03",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-04",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-05",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-06",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-07",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-09",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-10",region="RegionOne"} 48.0
nova_vcpus{aggregate="",hostname="compute-01",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-02",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-03",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-04",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-05",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-07",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-08",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-09",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-10",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-11",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-12",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-13",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-15",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-17",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-18",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-19",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-20",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-21",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-22",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-23",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-24",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-25",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-26",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-27",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-28",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-29",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-31",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-32",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-34",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-35",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-36",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-37",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-38",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-39",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-40",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-42",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-43",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-44",region="RegionOne"} 8.0
nova_vcpus{aggregate="",hostname="compute-45",region="RegionOne"} 8.0
# HELP nova_vcpus_used vcpus_used
# TYPE nova_vcpus_used gauge
nova_vcpus_used{aggregate="",hostname="compute-01",region="RegionOne"} 8.0
nova_vcpus_used{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-04",region="RegionOne"} 56.0
nova_vcpus_used{aggregate="",hostname="compute-05",region="RegionOne"} 8.0
nova_vcpus_used{aggregate="",hostname="compute-06",region="RegionOne"} 24.0
nova_vcpus_used{aggregate="",hostname="compute-07",region="RegionOne"} 41.0
nova_vcpus_used{aggregate="",hostname="compute-09",region="RegionOne"} 12.0
nova_vcpus_used{aggregate="",hostname="compute-10",region="RegionOne"} 21.0
nova_vcpus_used{aggregate="",hostname="compute-01",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-02",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-03",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-04",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-05",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-07",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-08",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-09",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-10",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-11",region="RegionOne"} 8.0
nova_vcpus_used{aggregate="",hostname="compute-12",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-13",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-15",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-17",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-18",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-19",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-20",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-21",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-22",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-23",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-24",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-25",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-26",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-27",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-28",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-29",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-31",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-32",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-34",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-35",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-36",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-37",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-38",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-39",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-40",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-42",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-43",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-44",region="RegionOne"} 0.0
nova_vcpus_used{aggregate="",hostname="compute-45",region="RegionOne"} 0.0
```

[buildstatus]: https://circleci.com/gh/niedbalski/openstack-exporter/tree/master.svg?style=shield
[circleci]: https://circleci.com/gh/niedbalski/openstack-exporter
[hub]: https://hub.docker.com/r/niedbalski/openstack-exporter/
