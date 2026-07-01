# ownstackctl

`ownstackctl` is the customer-visible Ownstack bootstrap CLI.

The source lives in this repository so customers can audit what the installer does before running it. The current implementation is intentionally transitional: `ownstackctl apply` validates the environment, emits structured progress events, and then delegates to the legacy shell installer at `scripts/setup_system_legacy.sh`.

Over time, the shell stages should move into Go behind the same CLI commands.

## Commands

```bash
ownstackctl plan
ownstackctl doctor
ownstackctl apply
ownstackctl status
```

- `plan` prints the bootstrap stages.
- `doctor` validates required environment variables and local prerequisites.
- `apply` runs the bootstrap.
- `status` is reserved for read-only cluster/service checks.

## Runtime Download

`setup_system.sh` remains the stable entrypoint used by Ownstack. It runs `ownstackctl` in this order:

1. `./bin/ownstackctl`, when a local binary exists.
2. `OWNSTACKCTL_URL`, when an explicit binary URL is provided.
3. `OWNSTACKCTL_VERSION`, when a pinned GitHub Release should be downloaded.
4. `go run ./cmd/ownstackctl`, when Go is installed.
5. The legacy shell installer as a fallback.

Pinned release example:

```bash
export OWNSTACKCTL_VERSION=ownstackctl-v0.1.0
./setup_system.sh
```

By default, releases are downloaded from `getownstack/ownstack-cluster-template`. Set `OWNSTACKCTL_REPO=owner/repo` to use another repository.

## Structured Progress

The CLI emits progress lines like:

```text
ownstack.event=stage_started id="platform" title="Platform install"
```

The web UI can consume these without parsing human-oriented shell output.
