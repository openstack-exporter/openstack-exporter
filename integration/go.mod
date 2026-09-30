module github.com/gophercloud/gophercloud/v2/openstack-exporter-integration

go 1.26.5

require (
	github.com/gophercloud/gophercloud/v2 v2.15.0
	github.com/gophercloud/utils/v2 v2.0.0-20260626221802-4ae35253ac13
	github.com/openstack-exporter/openstack-exporter v0.0.0
	github.com/prometheus/client_golang v1.24.1
	github.com/prometheus/client_model v0.6.3
	github.com/prometheus/common v0.71.0
)

require (
	github.com/alecthomas/kingpin/v2 v2.4.0 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gofrs/uuid/v5 v5.4.0 // indirect
	github.com/hashicorp/go-uuid v1.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gophercloud/gophercloud/v2 => ./gophercloud

replace github.com/openstack-exporter/openstack-exporter => ..
