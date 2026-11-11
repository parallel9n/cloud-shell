# Getting started

cloud-shell switches cloud CLI configuration by profile. A profile name is a label you choose for one customer. That label points AWS CLI, Azure CLI, gcloud, kubectl, Claude, and the .NET CLI at that customer's directory. Your normal `~/.zshrc` or `~/.bashrc` still starts the shell. The label shows up in the prompt (`customer Documents`).

In the examples, `customer` and `label` are two profile names. `cloud-shell customer` means "use the profile named `customer`." Any other valid name works the same way. They are not special commands.

A profile is a directory under `~/.cloud-profiles/<name>/`. Entering it sets the environment variables those tools already read. Credentials stay in that directory, so the next profile does not see them.

## Install

Go is required only if you build from this repository:

```sh
make test
make install
```

`make install` writes a static binary to `~/.local/bin/cloud-shell`. Put that directory on `PATH` if it is not already there.

A release binary does not need Go. Download the archive for your OS from the [GitHub releases](https://github.com/parallel9n/cloud-shell/releases), unpack it, and copy `cloud-shell` to a directory on `PATH`. `cloud-shell version` prints the release tag. A binary you built yourself prints `dev`.

## Create a profile

```sh
cloud-shell init customer
```

`customer` here is the label for that customer, and it becomes the profile name. That prints `profile customer ready at ~/.cloud-profiles/customer` and creates:

```
~/.cloud-profiles/customer/
  azure/                 AZURE_CONFIG_DIR
  aws/config             AWS_CONFIG_FILE
  aws/credentials        AWS_SHARED_CREDENTIALS_FILE
  gcloud/                CLOUDSDK_CONFIG
  kube/config            KUBECONFIG
  claude/                CLAUDE_CONFIG_DIR
  dotnet/                DOTNET_CLI_HOME
  dotnet/nuget/packages/ NUGET_PACKAGES
  shell/profile.zsh      sourced in zsh while this profile is active
  shell/profile.bashrc   sourced in bash while this profile is active
  shell/color            prompt color, default magenta
  env                    optional extra KEY=value lines
```

`init` creates missing files and leaves files that already exist. Running it again does not wipe credentials.

A profile name starts with a letter or digit and may contain letters, digits, `_`, `.`, and `-`, up to 64 characters. Command names are reserved: `complete`, `completion`, `current`, `env`, `help`, `hook`, `init`, and `list`.

## Sign in under that customer

The CLIs write where their environment variables point. Sign in from inside the profile, or the login lands in your default home config (`~/.aws`, `~/.azure`, and so on).

```sh
cloud-shell customer
aws configure
az login
gcloud auth login
kubectl config set-cluster ...
exit
```

`cloud-shell customer` starts a login shell (`$SHELL -l`, or `/bin/zsh` when `SHELL` is unset) with the customer paths set. On macOS and Linux that shell replaces the cloud-shell process. On Windows, cloud-shell waits until the shell exits. `exit` returns you to the shell you started from, with your usual config directories.

## Daily use

```sh
cloud-shell                         # list profiles; * marks the active one
cloud-shell customer                # login shell for that customer
cloud-shell label az account show   # one command, then back
cloud-shell current                 # print the active profile and its paths
eval "$(cloud-shell env customer)"  # export customer into this shell
```

`cloud-shell label az account show` sets the label environment and runs that command. It does not source `shell/profile.zsh`. Use the login shell, or `eval`, when you want the snippet and the prompt.

`eval "$(cloud-shell env customer)"` stays in effect until you eval another profile or open a new terminal. `cloud-shell current` with nothing selected prints `no profile active` and the `AZURE_CONFIG_DIR` still in effect (your default `~/.azure` when that variable is unset).

A missing name exits with an error and lists the profiles that do exist. `cloud-shell` with an empty `~/.cloud-profiles` reports that no profiles exist yet.

## Aliases

Short names live in `~/.config/cloud-shell/aliases`, one alias and one profile per line. `=` works the same way. Blank lines and `#` comments are ignored.

```
short customer
abbr label
```

`cloud-shell short` enters the `customer` directory. The prompt and `CLOUD_SHELL_PROFILE` use the directory name, `customer`. Completion offers both the directory names and the aliases.

## Prompt and completion

Add this to `~/.zshrc`. The `fpath` line goes before Oh My Zsh, so `compinit` sees the completion file. The hook goes at the end, after `prompt_dir` if you use an Agnoster-style prompt.

```sh
fpath=(${HOME}/.zsh/completions $fpath)
```

```sh
mkdir -p ~/.zsh/completions
cloud-shell completion zsh > ~/.zsh/completions/_cloud-shell
```

```sh
eval "$(cloud-shell hook zsh)"
```

Bash:

```sh
eval "$(cloud-shell hook bash)"
source <(cloud-shell completion bash)
```

The hook reads `CLOUD_SHELL_PROFILE`, then `CLOUD_PROFILE`. When one is set it sources that profile's `shell/profile.zsh` or `shell/profile.bashrc`. With Agnoster's `prompt_dir`, the customer is its own segment and the directory segment stays blue. Other zsh setups get the name prefixed on `PS1`, as in `(customer)`. With no profile selected, the prompt shows only the directory.

Set the segment color in `shell/color`. Accepted values are `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `turquoise`, a number from 0 to 255, or `#RRGGBB`. Anything else is magenta. `turquoise` is `#40E0D0`. The bash hook uses the named colors; a number or `#RRGGBB` falls back to magenta there. `init` writes `magenta`.

## Extra environment

`env` in the profile directory adds variables on top of the standard paths. Blank lines and `#` comments are ignored. Values may be bare, single-quoted, or double-quoted.

```
AWS_PROFILE=customer
AZURE_SUBSCRIPTION_ID=00000000-0000-0000-0000-000000000000
```

A key in this file replaces the same key from the standard set. `CLOUD_SHELL_PROFILE=other` in `env` would change the name the prompt shows.

## Where config lives

| Tool | Variable | Path under the profile |
| --- | --- | --- |
| Azure CLI | `AZURE_CONFIG_DIR` | `azure/` |
| AWS CLI | `AWS_CONFIG_FILE` | `aws/config` |
| AWS CLI | `AWS_SHARED_CREDENTIALS_FILE` | `aws/credentials` |
| gcloud | `CLOUDSDK_CONFIG` | `gcloud/` |
| kubectl | `KUBECONFIG` | `kube/config` |
| Claude | `CLAUDE_CONFIG_DIR` | `claude/` |
| .NET CLI | `DOTNET_CLI_HOME` | `dotnet/` |
| NuGet | `NUGET_PACKAGES` | `dotnet/nuget/packages/` |

`CLOUD_SHELL_PROFILE` and `CLOUD_PROFILE` are both set to the directory name. `aws/config` and `aws/credentials` are created with mode `0600`.

Override the locations with environment variables when the defaults are wrong:

| Variable | Default |
| --- | --- |
| `CLOUD_PROFILES_ROOT` | `~/.cloud-profiles` |
| `CLOUD_SHELL_ALIAS_FILE` | `~/.config/cloud-shell/aliases` |

## Commands

```
cloud-shell                         list profiles
cloud-shell list                    list profiles
cloud-shell init <profile>          create a profile directory
cloud-shell <profile>               login shell with that profile
cloud-shell <profile> <command...>  run one command with that profile
cloud-shell env <profile>           print POSIX export lines
cloud-shell current                 show the active profile
cloud-shell hook zsh|bash           print the prompt hook
cloud-shell completion zsh|bash     print the completion script
cloud-shell version                 print the version
cloud-shell help                    print usage
```

`cloud-shell --list` and `cloud-shell --current` are the same as `list` and `current`.
