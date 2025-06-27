# openstack-exporter snap

Snap published at <https://snapcraft.io/golang-openstack-exporter>

## Command

The command is provided in the snap as:

```sh
$ golang-openstack-exporter.openstack-exporter --help
usage: openstack-exporter [<flags>] [<cloud>]
...
```

This can be aliased to `openstack-exporter` if you wish:

```sh
sudo snap alias golang-openstack-exporter.openstack-exporter openstack-exporter
```

## Service

```sh
$ snap services golang-openstack-exporter.service
Service                            Startup   Current   Notes
golang-openstack-exporter.service  disabled  inactive  -
```

This service requires some configuration set first,
but then it can be managed as a snap service.
At least one of `cloud` or `multi-cloud` must be set,
and credentials will likely be required.

```sh
sudo snap set golang-openstack-exporter cloud=mycloud os-client-config=/etc/openstack/clouds.yaml
sudo snap start golang-openstack-exporter
```

### Logs

Logs for the service can be viewed by running:

```sh
sudo snap logs golang-openstack-exporter
```

## /etc/openstack access

To allow the snap to read from the `/etc/openstack/`, you may need to manually add the connection:

```sh
$ snap connections golang-openstack-exporter
Interface     Plug                                     Slot           Notes
home          golang-openstack-exporter:home           :home          -
network       golang-openstack-exporter:network        :network       -
network-bind  golang-openstack-exporter:network-bind   :network-bind  -
system-files  golang-openstack-exporter:etc-openstack  -              -

$ sudo snap connect golang-openstack-exporter:etc-openstack

$ snap connections golang-openstack-exporter
Interface     Plug                                     Slot           Notes
home          golang-openstack-exporter:home           :home          -
network       golang-openstack-exporter:network        :network       -
network-bind  golang-openstack-exporter:network-bind   :network-bind  -
system-files  golang-openstack-exporter:etc-openstack  :system-files  manual
```

## Configuration

The following config items are supported for the service.
Note that the service must be restarted for configuration changes to be applied.

```sh
$ sudo snap get -d golang-openstack-exporter
{
        "cloud": "",
        "collect-metric-time": "",
        "disable-cinder-agent-uuid": "",
        "disable-deprecated-metrics": "",
        "disable-metric": "",
        "disable-service": {
                "baremetal": "",
                "compute": "",
                "container-infra": "",
                "database": "",
                "dns": "",
                "gnocchi": "",
                "identity": "",
                "image": "",
                "load-balancer": "",
                "network": "",
                "object-store": "",
                "orchestration": "",
                "placement": "",
                "volume": ""
        },
        "disable-slow-metrics": "",
        "domain-id": "",
        "endpoint-type": "",
        "log": {
                "format": "",
                "level": ""
        },
        "multi-cloud": "",
        "os-client-config": "",
        "prefix": "",
        "web": {
                "listen-address": "",
                "telemetry-path": ""
        }
}
```

## Boolean options

These options must take a value of "true" or "false".
If set to "true", the corresponding flag is passed to the openstack-exporter.

- `collect-metric-time`
- `disable-cinder-agent-uuid`
- `disable-deprecated-metrics`
- `disable-service.baremetal`
- `disable-service.compute`
- `disable-service.container-infra`
- `disable-service.database`
- `disable-service.dns`
- `disable-service.gnocchi`
- `disable-service.identity`
- `disable-service.image`
- `disable-service.load-balancer`
- `disable-service.network`
- `disable-service.object-store`
- `disable-service.orchestration`
- `disable-service.placement`
- `disable-service.volume`
- `disable-slow-metrics`
- `multi-cloud`

Examples of configuring these options:

```sh
# turn on the option
sudo snap set golang-openstack-exporter multi-cloud=true
sudo snap set golang-openstack-exporter collect-metric-time=true
sudo snap set golang-openstack-exporter disable-service.dns=false

# turn off the option
sudo snap set golang-openstack-exporter disable-service.dns=false
# unsetting will reset to the default, which is false
sudo snap unset golang-openstack-exporter disable-slow-metrics
```

## Options with a value

These are options that take a value that is passed to the openstack-exporter cli:

- `cloud`
- `domain-id`
- `endpoint-type`
- `log.format`
- `log.level`
- `os-client-config`
- `prefix`
- `web.listen-address`
- `web.telemetry-path`

Examples of configuring these options:

```sh
# set a value
sudo snap set golang-openstack-exporter cloud=openstack
sudo snap set golang-openstack-exporter os-client-config=/etc/openstack/clouds.yaml

# revert to default
sudo snap set golang-openstack-exporter domain-id=
sudo snap unset golang-openstack-exporter log.format
```

### disable-metrics option

This option is similar to other options with a value,
but it takes multiple values, separated by whitespace.

- `disable-metrics`
