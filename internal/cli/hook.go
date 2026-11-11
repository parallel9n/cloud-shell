package cli

import "fmt"

const zshHook = `# cloud-shell prompt hook. Safe to eval more than once.
if [[ -z ${_CLOUD_SHELL_HOOK:-} ]]; then
  _CLOUD_SHELL_HOOK=1

  _cloud_shell_profile() {
    if [[ -n ${CLOUD_SHELL_PROFILE:-} ]]; then
      print -r -- "$CLOUD_SHELL_PROFILE"
      return
    fi
    if [[ -n ${CLOUD_PROFILE:-} ]]; then
      print -r -- "$CLOUD_PROFILE"
    fi
  }

  _cloud_shell_load_snippet() {
    local profile root snippet
    profile="$(_cloud_shell_profile)"
    [[ -n $profile ]] || return 0
    root="${CLOUD_PROFILES_ROOT:-$HOME/.cloud-profiles}/$profile"
    snippet="$root/shell/profile.zsh"
    if [[ -f $snippet ]]; then
      source "$snippet"
    fi
  }

  _cloud_shell_load_snippet

  # Color from <profile>/shell/color. Names, 0-255, #RRGGBB, or turquoise.
  # Anything else is magenta. turquoise is #40E0D0.
  _cloud_shell_profile_color() {
    local root color
    root="${CLOUD_PROFILES_ROOT:-$HOME/.cloud-profiles}/$1"
    color=""
    if [[ -f $root/shell/color ]]; then
      color="$(<"$root/shell/color")"
      color="${color//[[:space:]]/}"
      color="${(L)color}"
    fi
    case $color in
      turquoise) print -r -- '#40E0D0' ;;
      black|red|green|yellow|blue|magenta|cyan|white) print -r -- "$color" ;;
      [0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5]) print -r -- "$color" ;;
      \#[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]) print -r -- "$color" ;;
      *) print -r -- magenta ;;
    esac
  }

  if (( $+functions[prompt_dir] )); then
    prompt_dir() {
      local label
      label="$(_cloud_shell_profile)"
      if [[ -n $label ]]; then
        prompt_segment "$(_cloud_shell_profile_color "$label")" black " $label "
      fi
      prompt_segment "${AGNOSTER_DIR_BG:-blue}" "${AGNOSTER_DIR_FG}" '%1~'
    }
  else
    _cloud_shell_prefix_ps1() {
      local label
      label="$(_cloud_shell_profile)"
      [[ -n $label ]] || return 0
      case ${PS1:-} in
        "($label) "*) ;;
        *) PS1="($label) ${PS1:-}" ;;
      esac
    }
    autoload -Uz add-zsh-hook
    add-zsh-hook precmd _cloud_shell_prefix_ps1
  fi
fi
`

const bashHook = `# cloud-shell prompt hook. Safe to eval more than once.
if [ -z "${_CLOUD_SHELL_HOOK:-}" ]; then
  _CLOUD_SHELL_HOOK=1

  _cloud_shell_profile() {
    if [ -n "${CLOUD_SHELL_PROFILE:-}" ]; then
      printf '%s\n' "$CLOUD_SHELL_PROFILE"
      return
    fi
    if [ -n "${CLOUD_PROFILE:-}" ]; then
      printf '%s\n' "$CLOUD_PROFILE"
    fi
  }

  _cloud_shell_profile_color() {
    _root="${CLOUD_PROFILES_ROOT:-$HOME/.cloud-profiles}/$1"
    _color=""
    if [ -f "$_root/shell/color" ]; then
      _color=$(tr -d '[:space:]' < "$_root/shell/color" | tr '[:upper:]' '[:lower:]')
    fi
    case "$_color" in
      turquoise|black|red|green|yellow|blue|magenta|cyan|white) printf '%s\n' "$_color" ;;
      *) printf '%s\n' magenta ;;
    esac
  }

  _cloud_shell_ansi() {
    case "$1" in
      turquoise) printf '\033[48;2;64;224;208m\033[30m' ;;
      black) printf '\033[40;37m' ;;
      red) printf '\033[41;30m' ;;
      green) printf '\033[42;30m' ;;
      yellow) printf '\033[43;30m' ;;
      blue) printf '\033[44;37m' ;;
      cyan) printf '\033[46;30m' ;;
      white) printf '\033[47;30m' ;;
      *) printf '\033[45;30m' ;;
    esac
  }

  _profile="$(_cloud_shell_profile)"
  if [ -n "$_profile" ]; then
    _root="${CLOUD_PROFILES_ROOT:-$HOME/.cloud-profiles}/$_profile"
    _snippet="$_root/shell/profile.bashrc"
    if [ -f "$_snippet" ]; then
      . "$_snippet"
    fi
    if [ "${_CLOUD_SHELL_PS1_LABEL:-}" != "$_profile" ]; then
      _CLOUD_SHELL_PS1_LABEL="$_profile"
      _color="$(_cloud_shell_profile_color "$_profile")"
      PS1="\[$(_cloud_shell_ansi "$_color")\] ${_profile} \[\033[0m\] ${PS1:-}"
    fi
  fi
  unset _profile _root _snippet _color
fi
`

const zshCompletion = `#compdef cloud-shell

local -a profiles subcmds
profiles=(${(f)"$(command cloud-shell complete --profiles 2>/dev/null)"})
subcmds=(
  'env:print export lines for a profile'
  'init:create a profile directory'
  'current:show the active profile'
  'list:list profiles'
  'completion:print a shell completion script'
  'hook:print the prompt hook'
)

if (( CURRENT == 2 )); then
  _describe -t commands 'command' subcmds
  _describe -t profiles 'profile' profiles
  return
fi

case ${words[2]} in
  env|init)
    if (( CURRENT == 3 )); then
      _describe -t profiles 'profile' profiles
    fi
    ;;
  completion|hook)
    if (( CURRENT == 3 )); then
      _values 'shell' zsh bash
    fi
    ;;
  *)
    if (( CURRENT == 3 )); then
      _command_names -e
    else
      _files
    fi
    ;;
esac
`

const bashCompletion = `_cloud_shell_complete() {
  local cur prev profiles
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  profiles="$(command cloud-shell complete --profiles 2>/dev/null)"
  case "$prev" in
    env|init)
      COMPREPLY=( $(compgen -W "$profiles" -- "$cur") )
      return 0
      ;;
    completion|hook)
      COMPREPLY=( $(compgen -W "zsh bash" -- "$cur") )
      return 0
      ;;
  esac
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "env init current list completion hook --list --current --help $profiles" -- "$cur") )
  fi
}
complete -F _cloud_shell_complete cloud-shell
`

// Hook returns the shell snippet that labels the path and sources the profile rc.
func Hook(shell string) (string, error) {
	switch shell {
	case "zsh":
		return zshHook, nil
	case "bash":
		return bashHook, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (want zsh or bash)", shell)
	}
}

// Completion returns a completion script for shell.
func Completion(shell string) (string, error) {
	switch shell {
	case "zsh":
		return zshCompletion, nil
	case "bash":
		return bashCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (want zsh or bash)", shell)
	}
}
