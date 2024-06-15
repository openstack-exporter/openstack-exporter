package openstack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type LoadbalancerTestSuite struct {
	BaseOpenStackTestSuite
}

var loadbalancerExpectedUp = `                       
# HELP openstack_loadbalancer_pool_status pool_status
# TYPE openstack_loadbalancer_pool_status gauge
openstack_loadbalancer_pool_status{id="ca00ed86-94e3-440e-95c6-ffa35531081e",lb_algorithm="ROUND_ROBIN",loadbalancers="e7284bb2-f46a-42ca-8c9b-e08671255125",name="my_test_pool",operating_status="ERROR",project_id="8b1632d90bfe407787d9996b7f662fd7",protocol="TCP",provisioning_status="ACTIVE"} 0
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
# HELP openstack_loadbalancer_total_pools total_pools
# TYPE openstack_loadbalancer_total_pools gauge
openstack_loadbalancer_total_pools 1
# HELP openstack_loadbalancer_up up
# TYPE openstack_loadbalancer_up gauge
openstack_loadbalancer_up 1
`

func (suite *LoadbalancerTestSuite) TestLoadbalancerExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(loadbalancerExpectedUp))
	assert.NoError(suite.T(), err)
}
