package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type KeystoneTestSuite struct {
	BaseOpenStackTestSuite
}

var keystoneExpectedUp = `                       
# HELP openstack_identity_domains domains
# TYPE openstack_identity_domains gauge
openstack_identity_domains 1
# HELP openstack_identity_domain_info domain_info
# TYPE openstack_identity_domain_info gauge
openstack_identity_domain_info{description="Owns users and tenants (i.e. projects) available on Identity API v2.",enabled="true",id="default",name="Default"} 1
# HELP openstack_identity_groups groups
# TYPE openstack_identity_groups gauge
openstack_identity_groups 2
# HELP openstack_identity_project_info project_info
# TYPE openstack_identity_project_info gauge
openstack_identity_project_info{description="",domain_id="1bc2169ca88e4cdaaba46d4c15390b65",enabled="true",id="4b1eb781a47440acb8af9850103e537f",is_domain="false",name="swifttenanttest4",parent_id="",tags=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="0c4e939acacf4376bdcd1129f1a054ad",is_domain="false",name="admin",parent_id="",tags=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="2db68fed84324f29bb73130c6c2094fb",is_domain="false",name="swifttenanttest2",parent_id="",tags=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="3d594eb0f04741069dbbb521635b21c7",is_domain="false",name="service",parent_id="",tags=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="43ebde53fc314b1c9ea2b8c5dc744927",is_domain="false",name="swifttenanttest1",parent_id="",tags=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="5961c443439d4fcebe42643723755e9d",is_domain="false",name="invisible_to_admin",parent_id="",tags=""} 1
openstack_identity_project_info{description="",domain_id="default",enabled="true",id="fdb8424c4e4f4c0ba32c52e2de3bd80e",is_domain="false",name="alt_demo",parent_id="",tags=""} 1
openstack_identity_project_info{description="This is a demo project.",domain_id="default",enabled="true",id="0cbd49cbf76d405d9c86562e1d579bd3",is_domain="false",name="demo",parent_id="",tags=""} 1
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
`

func (suite *KeystoneTestSuite) TestKeystoneExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(keystoneExpectedUp))
	assert.NoError(suite.T(), err)
}
