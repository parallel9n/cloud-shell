# cloud-shell

One static binary that points AWS, gcloud, Azure, kubectl, Claude, and .NET at a directory per customer. Shell startup stays your normal `~/.zshrc`. The active profile is drawn in the path segment (`hex Documents`).

Profiles live in `~/.cloud-profiles/<name>/`:

```
azure/          AZURE_CONFIG_DIR
aws/config      AWS_CONFIG_FILE
aws/credentials AWS_SHARED_CREDENTIALS_FILE
gcloud/         CLOUDSDK_CONFIG
kube/config     KUBECONFIG
claude/         CLAUDE_CONFIG_DIR
dotnet/         DOTNET_CLI_HOME and NUGET_PACKAGES
shell/          optional profile.zsh, profile.bashrc, and color
env             optional KEY=value extras
```

## Install

```sh
make test
make install
```

`make install` copies a static binary to `~/.local/bin/cloud-shell`. Go is only required to build.

## Use

```sh
cloud-shell                  # list profiles (* marks the active one)
cloud-shell init acme
cloud-shell hex              # login shell with Hexagon paths
cloud-shell hex az account show
eval "$(cloud-shell env hex)"
cloud-shell current
```

Aliases such as `hexagon` → `hex` are lines in `~/.config/cloud-shell/aliases`:

```
hexagon hex
hxr hex
vik viking
```

## Prompt and completion

```sh
# before Oh My Zsh is sourced, so compinit sees it
fpath=(${HOME}/.zsh/completions $fpath)

# at the end of ~/.zshrc, after prompt_dir
eval "$(cloud-shell hook zsh)"
```

```sh
mkdir -p ~/.zsh/completions
cloud-shell completion zsh > ~/.zsh/completions/_cloud-shell
```

Bash:

```sh
eval "$(cloud-shell hook bash)"
source <(cloud-shell completion bash)
```

The hook reads `CLOUD_SHELL_PROFILE` or `CLOUD_PROFILE`. The profile is its own prompt segment, colored from `shell/color` (`black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `turquoise`, a number 0-255, or `#RRGGBB`). The directory segment stays blue. With no profile selected, only the directory is shown.

## Releases

Pushes to `main` that follow [Conventional Commits](https://www.conventionalcommits.org/) open a release-please pull request. Merging that pull request tags the release and attaches static binaries:

- macOS (`darwin/amd64`, `darwin/arm64`)
- Linux (`linux/amd64`, `linux/arm64`)
- Windows (`windows/amd64`, `windows/arm64`)

`cloud-shell version` prints the release tag. A local build prints `dev`.
