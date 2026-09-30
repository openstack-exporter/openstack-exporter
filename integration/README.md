# Integration helpers

The tests use Gophercloud acceptance helpers copied into the existing `clients`,
`funcs`, and `tools` packages. These are ordinary committed Go files, so GitHub
shows their contents and update diffs without submodules or a separate repository.
Gophercloud's `internal` packages cannot be imported directly by this module.
See [the upstream discussion](https://github.com/gophercloud/gophercloud/issues/3501).

`GOPHERCLOUD_REVISION` pins the source commit (initially Gophercloud v2.15.0).
`LICENSE.gophercloud` preserves the upstream license and copyright notices.
Each `*_generated.go` file records its upstream source path and commit.

## Updating

From the repository root, run:

```sh
go run ./script/sync-gophercloud
go test ./script/sync-gophercloud
go test -short ./integration/...
```

To update, replace `integration/GOPHERCLOUD_REVISION` with a full upstream commit
SHA compatible with the Gophercloud version in `go.mod`, run those commands, and
review and commit the diff. The updater fetches only that commit into a temporary
checkout. With an existing clean checkout at the pinned commit, skip the fetch:

```sh
go run ./script/sync-gophercloud -source /path/to/gophercloud
```

The selection list in `script/sync-gophercloud/main.go` names the functions and
types to copy, including their helper dependencies. It copies declaration bodies
and comments, rewrites acceptance imports to this module, and removes unused
imports. Missing selected declarations cause an error before any files are written.
Run it twice to check that the second run produces no additional diff.

## Local behavior

Edit the ordinary source files for exporter-specific behavior; do not edit the
`*_generated.go` files. Authentication stays in `clients/clients.go` to retain the
30-second authentication deadline, region selection, and redacted debug logging.
Only service constructors used by the suite are retained.

Networking keeps its request and cleanup deadlines, registered cleanup, and VPNaaS
configuration. Image creation keeps queued images without uploading data. Volume
helpers keep status checks and wait for snapshots and backups to disappear before
volume cleanup. These behaviors differ from upstream and are maintained locally.

Short tests compile the full suite without creating cloud resources. To exercise
live behavior, run the existing service workflows or `script/integration-suites`
commands with a configured OpenStack deployment.
