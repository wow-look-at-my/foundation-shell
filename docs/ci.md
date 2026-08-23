# CI

`.github/workflows/ci.yml` runs two jobs on every push.

## `go` — build, vet, test, coverage

Runs `wow-look-at-my/go-toolchain@v1` with `working-directory: src`.

**Why `working-directory: src`.** The Go module lives in `src/`, not the repo
root. go-toolchain's matrix command reads `go.mod` from its working directory
— there is no module auto-discovery on this path — builds into
`<working-directory>/build/`, and the action uploads that same directory as
the artifact.

An initialized `spec/` submodule contains its own `generator/go.mod`, which is
the other reason a repo-root run is wrong: it would walk into the submodule.

**Why the `permissions:` block is what it is.** go-toolchain's autorelease is
on by default and publishes branch builds to buildhost. It verifies up front
that the workflow grants:

| Permission | Needed for |
|------------|-----------|
| `id-token: write` | OIDC, to authenticate to buildhost |
| `contents: write` | the dependency-graph snapshot go-toolchain submits; GitHub rejects it with HTTP 403 under `contents: read` |
| `actions: read` | artifact download, and the embedded no-all-builds guard's job scan |
| `checks: read` | the same guard's check-run scan |

Autorelease fails the build if `id-token: write` or `actions: read` is
missing, rather than skipping quietly.

A job-level `permissions:` block REPLACES the workflow-level one, so a job
that declares its own must list all of these again.

## `bats` — conformance suite

Installs `just` and `bats`, runs `just build`, then `bats tests/`.

`just` is installed from a pinned release rather than from apt: the justfile
uses the `[working-directory: '...']` attribute, which needs just 1.38 or
newer, and the apt package is older than that on the runner image.

## Comment walls fail the build

go-toolchain embeds `wow-look-at-my/actions@yaml-comment-block#latest`, which
rejects more than ONE comment line in a row in workflow and action YAML.
Blank lines between comment lines do not split a block. This file is where
that displaced prose goes; the workflow keeps a one-line pointer to it.

## Not used here

The repo does NOT fetch the `spec/` submodule in CI (`actions/checkout`
default). Keep it that way unless CI starts reading spec content.
