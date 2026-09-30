# Integration helpers

The integration suite uses four Git subtrees from
[Gophercloud](https://github.com/gophercloud/gophercloud), initially at v2.15.0
(commit `4aae510cb4d2b51a46151542c7fbca55f8952f04`).

| Local directory | Upstream directory |
| --- | --- |
| `integration/clients` | `internal/acceptance/clients` |
| `integration/tools` | `internal/acceptance/tools` |
| `integration/compute` | `internal/acceptance/openstack/compute/v2` |
| `integration/baremetal` | `internal/acceptance/openstack/baremetal/v1` |

These are Git subtree imports with `--squash`. Git records each imported subtree
commit in the merge history with `git-subtree-dir` and `git-subtree-split` trailers.
Updates use Git merges, which preserve local adaptations and report conflicts.
The files appear as normal browsable directories on GitHub. There is no generated
copying script, and cloning this repository needs no extra checkout step.
`LICENSE.gophercloud` preserves the upstream license and copyright notices.

## Updating a subtree

Install Git's `subtree` command if your Git distribution does not include it.
Start with a clean exporter checkout and clone the upstream source once:

```sh
git clone https://github.com/gophercloud/gophercloud.git /tmp/gophercloud
```

Check out the desired upstream commit, keeping it compatible with `go.mod`.
From the exporter repository root, update a subtree by splitting its upstream
folder and pulling that Git branch. For example, to update tools:

```sh
git -C /tmp/gophercloud checkout <upstream-commit>
git -C /tmp/gophercloud subtree split \
  --prefix=internal/acceptance/tools -b exporter-tools
git subtree pull --prefix=integration/tools \
  /tmp/gophercloud exporter-tools --squash
```

Use the corresponding paths in the table for the other packages, and a distinct
split branch for each. Use a fresh split branch name on subsequent updates.
Resolve any merge conflicts, update the source revision recorded above, then run:

```sh
go test -short -tags 'fixtures acceptance' ./integration/...
```

Review the resulting changes as a normal PR. Keep upstream `*_test.go` files
removed: the exporter runs its own integration scenarios, and upstream test files
import additional internal packages outside these subtrees.

## Local adaptations

The subtree import commits contain the upstream files. Subsequent commits record
local adaptations: rewriting internal helper imports, removing upstream tests,
and retaining the shared 30-second authentication deadline, region selection, and
redacted debug logging. Client constructors unused by the exporter are removed.
Files with adapted imports or authentication carry a comment explaining the change.

Compute and baremetal tests import their subtree packages directly. Networking,
image, and volume behavior stays in `integration/funcs`, using the shared clients
and tools. This retains bounded networking cleanup, queued-image creation, and
volume deletion waits without duplicate compute or baremetal helper copies.
