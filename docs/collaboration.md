# Collaboration

cloud-shell is one command. Changes stay small, and the public surface stays the binary.

By contributing, you agree that your changes are licensed under the [MIT License](../LICENSE).

## Talk first

Open an issue before a large change. A bug report should say what you ran, what you expected, and the output. A feature request should name the CLI it would point at a profile directory, and the environment variable that CLI already reads.

## Make a change

```sh
make test
```

`make test` runs `go test ./...`. Continuous integration runs that on Linux, macOS, and Windows, and it builds the release archives with `scripts/release-assets.sh`.

Format Go with `gofmt` before you push. `make fmt` does that.

`main.go` is the only package outside `internal`. Packages under `internal` stay private to this module. [layout.md](layout.md) explains that split.

## Examples

Profile names in docs, help text, and tests are labels, not real customers. Use `customer` and `label`. An alias in an example is `short` for `customer` or `abbr` for `label`.

Do not commit credentials, a populated `~/.cloud-profiles` tree, or tokens.

## Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/). release-please reads them on `main` and opens the release pull request.

```
feat: add a profile color for the prompt
fix: resolve executables on Windows
docs: describe profile names as labels
```

Use `feat` for something new, `fix` for a broken behavior, and `docs` for documentation only. The subject line says why the change exists, in the imperative.

## Pull requests

One change per pull request. Describe what a user can do after it merges, and how you tested it. Fill the test plan with the commands you ran.

Wait for the Linux, macOS, and Windows test jobs. The `dist` job must build every archive.

## Reviews

Review the behavior a user sees: the profile directory, the environment variables, and the prompt. Ask when a name in an example looks like a real customer. Leave credentials and private profile contents out of review comments.
