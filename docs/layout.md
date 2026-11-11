# Layout

cloud-shell is one command. The repository holds that command, its tests, and the release build.

```
main.go                      process entry
internal/cli                 arguments, prompt hook, process start
internal/profile             profile directories and environment
scripts/release-assets.sh    cross-compiled release archives
Makefile                     build, test, install, dist
.github/workflows            test, dist check, GitHub release
```

## Why the code is in `internal`

Go treats a directory named `internal` as private to the tree that contains it. Another module cannot import `github.com/wcampos/cloud-shell/internal/cli` or `github.com/wcampos/cloud-shell/internal/profile`. The compiler rejects the import.

This repository publishes a static binary, not a Go library. `main.go` is the only package meant to be built and installed. `cli` and `profile` are split out so tests can call them directly, and so argument handling stays separate from the on-disk profile layout. They live under `internal` so those packages do not look like an API for other modules. Moving a type out of `internal` would be a decision to support that import.

Inside this module the imports are allowed. `main` imports `internal/cli`. `internal/cli` imports `internal/profile`. Both importers sit in the module that owns `internal`.

## Packages

`main.go` calls `cli.Main` and exits with its status code. Release builds set `main.version` with `-ldflags`. A local build leaves the variable as `dev`, which is what `cloud-shell version` prints.

`internal/cli` parses arguments, writes `export` lines, and starts a login shell or a single command with the profile environment. `hook.go` is the zsh and bash prompt snippet printed by `cloud-shell hook`. `exec_unix.go` replaces the current process with `syscall.Exec`. `exec_other.go` is the Windows build: Windows has no `Exec`, so the command runs as a child and cloud-shell waits for it. `look.go` finds the executable on `PATH`, including Windows `PATHEXT`.

`internal/profile` maps a customer name to `~/.cloud-profiles/<name>/` and to the environment variables AWS, Azure, gcloud, kubectl, Claude, and .NET read. It also reads `~/.config/cloud-shell/aliases` and the optional per-profile `env` file. `cloud-shell init` creates that tree through `Store.Init`. Existing credential files are left in place.

Tests sit next to the code they cover: `internal/cli/cli_test.go` and `internal/profile/profile_test.go`. `go test ./...` runs both.

## What `scripts/` is for

Nothing in `scripts/` is part of the installed binary. The shell does not source these files, and `cloud-shell` does not call them.

`scripts/release-assets.sh` cross-compiles a static binary for each target, packs it, and writes checksums:

| Target | Archive |
| --- | --- |
| `darwin/amd64`, `darwin/arm64` | `.tar.gz` |
| `linux/amd64`, `linux/arm64` | `.tar.gz` |
| `windows/amd64`, `windows/arm64` | `.zip` |

Each archive is named `cloud-shell_<version>_<os>_<arch>`. The script also writes `SHA256SUMS` in the output directory. Builds set `CGO_ENABLED=0`, so the binaries do not link a system C library. The version argument is passed through to `main.version`. A leading `v` is stripped from the archive name, so tag `v0.1.0` produces `cloud-shell_0.1.0_linux_amd64.tar.gz`.

Three callers use that script:

- `make dist` runs `bash scripts/release-assets.sh $(VERSION) dist` and leaves the archives in `dist/`. `VERSION` defaults to `dev`.
- `.github/workflows/ci.yml` runs the same script with version `dev` on every push to `main` and every pull request, then uploads `dist/*` as a workflow artifact. That job checks that every target still builds. It does not publish a GitHub release.
- `.github/workflows/release-please.yml` runs the script with the release tag after release-please tags `main`, then `gh release upload` attaches `dist/*` to that release.

`make build` and `make install` compile for the machine they run on. They do not use `scripts/`.
