# Integration tests

`gophercloud/` is a Git submodule pointing directly to
[Gophercloud](https://github.com/gophercloud/gophercloud). GitHub displays that
folder as a link to the pinned upstream commit. The initial pin is v2.15.0
(`4aae510cb4d2b51a46151542c7fbca55f8952f04`), matching the SDK version in the
exporter's `go.mod`. Upstream source and license remain in the submodule.

## Running tests

Initialize the submodule after cloning, then run tests from this module:

```sh
git submodule update --init integration/gophercloud
go -C integration test -short -tags 'fixtures acceptance' ./...
```

Alternatively, clone the exporter with `git clone --recurse-submodules`.
Short mode checks the integration code without creating OpenStack resources.
For live tests, use the existing `script/integration-suites` commands with a
configured DevStack installation. CI initializes the submodule automatically.
The exporter unit tests still run from the repository root.

## Go module layout

Integration tests have their own module because Go restricts imports of
Gophercloud's `internal/acceptance` packages. Its module path is
`github.com/gophercloud/gophercloud/v2/openstack-exporter-integration` to place the
importing packages under the prefix required by that rule. This is a local test
module maintained in the exporter repository. It is not published by Gophercloud.

Local replacements resolve the SDK to `./gophercloud` and the exporter to `..`.
The tests therefore use the pinned submodule source and the current exporter
checkout. The production module continues to use its existing SDK dependency.

Baremetal helpers, compute waits and deletes, environment choices, and shared
tools are imported directly from the submodule. Local code retains authentication
with a 30-second deadline and redacted debug logging, along with exporter-specific
networking, image, and volume setup. Compute creation stays local so its Neutron
lookup uses the bounded, redacted client; shared waiting and deletion come from
upstream. There are no copied helper directories or generated update scripts.

## Updating Gophercloud

Check out the desired upstream tag or commit, keeping it compatible with the
SDK version in the exporter's `go.mod`:

```sh
git -C integration/gophercloud fetch origin tag v2.15.0
git -C integration/gophercloud checkout --detach v2.15.0
go -C integration mod tidy
go -C integration test -short -tags 'fixtures acceptance' ./...
git add integration/gophercloud integration/go.mod integration/go.sum
```

Use the new version in the commands and update the module's SDK requirement when
upgrading. Commit the submodule pointer and any module changes in the same PR.
