# CI

`.github/workflows/ci.yml` runs one job on every push.

## `go` — build, vet, test, coverage, conformance

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
| `deployments: write` | the GitHub Deployment autorelease registers for each publish |
| `artifact-metadata: write` | recording the upload on the org's linked-artifacts page |

Every one of these is a hard failure when missing, not a skipped step.
Registering the Deployment and recording the artifact are both part of
publishing and have no opt-out; the only way to avoid needing the last two is
`autorelease: 'false'`.

A job-level `permissions:` block REPLACES the workflow-level one, so a job
that declares its own must list all of these again.

**Why `os` and `arch` are set explicitly.** go-toolchain's matrix defaults to
ONE Cosmopolitan fat APE covering `--cosmo-platforms`. This shell cannot be
built that way: it depends on `github.com/chzyer/readline`, whose terminal
handling lives entirely in files gated on a real GOOS. Under Cosmopolitan's
GOOS none of them are selected, and the build dies on `undefined: State`,
`undefined: GetScreenWidth`, `undefined: SuspendMe` and the rest.

Naming either `os` or `arch` switches the matrix back to the cartesian
product of native per-platform binaries, which readline compiles for cleanly
— all 18 (3 commands x 3 OSes x 2 arches) build. That is also what this repo
published before the default changed; it never opted into cosmo.

Turning cosmo back on means replacing readline first.

**Why the job installs bubblewrap.** go-toolchain runs `src/dats/*.dats` — the
conformance suite — after every build, and dats sandboxes each command. Its
`auto` backend prefers bubblewrap and falls back to docker, which would run the
commands inside an image that has neither the built binaries nor the coreutils
the suites call. The runner image ships no bubblewrap, so the job installs it.

There is no separate conformance job: the suites are part of the same build,
and a suite failure fails it.

## Comment walls fail the build

go-toolchain embeds `wow-look-at-my/actions@yaml-comment-block#latest`, which
rejects more than ONE comment line in a row in workflow and action YAML.
Blank lines between comment lines do not split a block. This file is where
that displaced prose goes; the workflow keeps a one-line pointer to it.

## Not used here

The repo does NOT fetch the `spec/` submodule in CI (`actions/checkout`
default). Keep it that way unless CI starts reading spec content.
