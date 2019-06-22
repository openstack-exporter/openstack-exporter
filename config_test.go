package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigAuthVerify(t *testing.T) {
	const testVerifiedConnectionConfig = `
clouds:
 test.cloud:
   region_name: RegionOne
   identity_api_version: 3
   identity_interface: internal
   auth:
     username: 'admin'
     password: 'admin'
     project_name: 'admin'
     project_domain_name: 'Default'
     user_domain_name: 'Default'
     auth_url: 'http://test.cloud:35357/v3'
`

	const testUnverifiedConnectionConfig = `
clouds:
 test.cloud:
   region_name: RegionOne
   identity_api_version: 3
   identity_interface: internal
   auth:
     username: 'admin'
     password: 'admin'
     project_name: 'admin'
     project_domain_name: 'Default'
     user_domain_name: 'Default'
     auth_url: 'http://test.cloud:35357/v3'
     verify: false
`

	cfg, err := NewCloudConfigFromByteArray([]byte(testVerifiedConnectionConfig))
	if assert.NoError(t, err) {
		cloud, err := cfg.GetByName("test.cloud")
		if assert.NoError(t, err) {
			assert.True(t, cloud.Auth.Verify)
		}
	}

	cfg, err = NewCloudConfigFromByteArray([]byte(testUnverifiedConnectionConfig))
	if assert.NoError(t, err) {
		cloud, err := cfg.GetByName("test.cloud")
		if assert.NoError(t, err) {
			assert.False(t, cloud.Auth.Verify)
		}
	}
}
