// Package cli is the cloud-shell command line.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wcampos/cloud-shell/internal/profile"
)

const usage = `Usage: cloud-shell [profile] [command...]
       cloud-shell env <profile>
       cloud-shell init <profile>
       cloud-shell current
       cloud-shell completion zsh|bash
       cloud-shell hook zsh|bash
       cloud-shell --list
       cloud-shell --current
       cloud-shell version

Profiles live under $CLOUD_PROFILES_ROOT (default ~/.cloud-profiles).
Aliases live in ~/.config/cloud-shell/aliases.

Examples (customer and label are profile names you choose):
  cloud-shell customer
  cloud-shell label az account show
  eval "$(cloud-shell env customer)"
  cloud-shell init customer
`

// Runner executes cloud-shell commands.
// A zero Stdout or Stderr uses the process standard streams.
// A nil Exec replaces the current process.
type Runner struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Store   profile.Store
	Exec    func(argv0 string, argv []string, env []string) error
	Look    func(file string, env []string) (string, error)
	Shell   func() string
	Environ func() []string
	Version string
}

// Main runs cloud-shell with the process arguments and streams.
func Main(version string) int {
	r, err := New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cloud-shell: %v\n", err)
		return 1
	}
	r.Version = version
	return r.Run(os.Args[1:])
}

// New returns a Runner using the default profile store.
func New() (Runner, error) {
	store, err := profile.Default()
	if err != nil {
		return Runner{}, err
	}
	return Runner{Store: store}, nil
}

// Run executes one invocation. args excludes the program name.
func (r Runner) Run(args []string) int {
	if len(args) == 0 {
		return r.list()
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(r.stdout(), usage)
		return 0
	case "-l", "--list", "list":
		if len(args) != 1 {
			return r.usageErr("list takes no arguments")
		}
		return r.list()
	case "-c", "--current", "current":
		if len(args) != 1 {
			return r.usageErr("current takes no arguments")
		}
		return r.current()
	case "env":
		if len(args) != 2 {
			return r.usageErr("env requires a profile")
		}
		return r.env(args[1])
	case "init":
		if len(args) != 2 {
			return r.usageErr("init requires a profile name")
		}
		return r.initProfile(args[1])
	case "completion":
		if len(args) != 2 {
			return r.usageErr("completion requires zsh or bash")
		}
		return r.completion(args[1])
	case "hook":
		if len(args) != 2 {
			return r.usageErr("hook requires zsh or bash")
		}
		return r.hook(args[1])
	case "version", "--version":
		fmt.Fprintf(r.stdout(), "cloud-shell %s\n", r.versionString())
		return 0
	case "complete":
		if len(args) == 2 && args[1] == "--profiles" {
			return r.completeProfiles()
		}
		return r.usageErr("complete requires --profiles")
	default:
		if strings.HasPrefix(args[0], "-") {
			return r.usageErr("unknown flag " + args[0])
		}
		return r.enter(args[0], args[1:])
	}
}

func (r Runner) list() int {
	store, err := r.store()
	if err != nil {
		return r.fail(err)
	}
	names, err := store.List()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(r.stderr(), "No profiles yet under %s\n", store.Root)
			return 1
		}
		return r.fail(err)
	}
	active := r.activeName(store)
	for _, name := range names {
		if name == active {
			fmt.Fprintf(r.stdout(), "* %s\n", name)
			continue
		}
		fmt.Fprintf(r.stdout(), "  %s\n", name)
	}
	return 0
}

func (r Runner) current() int {
	name := profile.Active()
	if name == "" {
		azure := os.Getenv("AZURE_CONFIG_DIR")
		if azure == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return r.fail(fmt.Errorf("home directory: %w", err))
			}
			azure = filepath.Join(home, ".azure")
		}
		fmt.Fprintf(r.stdout(), "cloud-shell: no profile active (AZURE_CONFIG_DIR=%s)\n", azure)
		return 0
	}
	store, err := r.store()
	if err != nil {
		return r.fail(err)
	}
	vars, err := store.Describe(name)
	if err != nil {
		fmt.Fprintf(r.stdout(), "cloud-shell profile: %s\n", name)
		return r.fail(err)
	}
	writeVars(r.stdout(), labelFrom(vars, name), vars)
	return 0
}

func (r Runner) activeName(store profile.Store) string {
	name := profile.Active()
	if name == "" {
		return ""
	}
	canonical, err := store.Canonical(name)
	if err != nil {
		return name
	}
	return canonical
}

func labelFrom(vars []profile.Var, fallback string) string {
	for _, v := range vars {
		if v.Key == "CLOUD_SHELL_PROFILE" && v.Value != "" {
			return v.Value
		}
	}
	return fallback
}

func (r Runner) env(name string) int {
	store, err := r.store()
	if err != nil {
		return r.fail(err)
	}
	text, err := store.Exports(name)
	if err != nil {
		return r.fail(err)
	}
	fmt.Fprint(r.stdout(), text)
	return 0
}

func (r Runner) initProfile(name string) int {
	store, err := r.store()
	if err != nil {
		return r.fail(err)
	}
	lay, err := store.Init(name)
	if err != nil {
		return r.fail(err)
	}
	fmt.Fprintf(r.stdout(), "profile %s ready at %s\n", lay.Name, lay.Root)
	return 0
}

func (r Runner) completion(shell string) int {
	text, err := Completion(shell)
	if err != nil {
		return r.fail(err)
	}
	fmt.Fprint(r.stdout(), text)
	return 0
}

func (r Runner) hook(shell string) int {
	text, err := Hook(shell)
	if err != nil {
		return r.fail(err)
	}
	fmt.Fprint(r.stdout(), text)
	return 0
}

func (r Runner) completeProfiles() int {
	store, err := r.store()
	if err != nil {
		return r.fail(err)
	}
	names, err := store.Names()
	if err != nil {
		return r.fail(err)
	}
	for _, name := range names {
		fmt.Fprintln(r.stdout(), name)
	}
	return 0
}

func (r Runner) enter(name string, args []string) int {
	store, err := r.store()
	if err != nil {
		return r.fail(err)
	}
	canonical, err := store.Resolve(name)
	if err != nil {
		return r.fail(err)
	}
	vars, err := store.Vars(canonical)
	if err != nil {
		return r.fail(err)
	}
	env := profile.MergeEnv(r.environ(), vars)
	if len(args) == 0 {
		shell := r.shell()
		fmt.Fprintf(r.stderr(), "cloud-shell: entered profile '%s'\n", canonical)
		writeVars(r.stderr(), canonical, vars)
		if err := r.exec(shell, []string{shell, "-l"}, env); err != nil {
			fmt.Fprintf(r.stderr(), "cloud-shell: exec %s: %v\n", shell, err)
			return 1
		}
		return 0
	}
	path, err := r.look(args[0], env)
	if err != nil {
		fmt.Fprintf(r.stderr(), "cloud-shell: %v\n", err)
		return 1
	}
	if err := r.exec(path, args, env); err != nil {
		fmt.Fprintf(r.stderr(), "cloud-shell: exec %s: %v\n", path, err)
		return 1
	}
	return 0
}

func (r Runner) fail(err error) int {
	fmt.Fprintf(r.stderr(), "cloud-shell: %v\n", err)
	var missing *profile.NotFoundError
	if errors.As(err, &missing) {
		r.printAvailable()
	}
	return 1
}

func (r Runner) printAvailable() {
	store, err := r.store()
	if err != nil {
		return
	}
	names, err := store.List()
	if err != nil || len(names) == 0 {
		return
	}
	fmt.Fprintln(r.stderr(), "available:")
	for _, name := range names {
		fmt.Fprintf(r.stderr(), "  %s\n", name)
	}
}

func (r Runner) usageErr(msg string) int {
	fmt.Fprintf(r.stderr(), "cloud-shell: %s\n", msg)
	fmt.Fprint(r.stderr(), usage)
	return 2
}

func (r Runner) store() (profile.Store, error) {
	if r.Store.Root != "" {
		return r.Store, nil
	}
	return profile.Default()
}

func (r Runner) versionString() string {
	if r.Version != "" {
		return r.Version
	}
	return "dev"
}

func (r Runner) stdout() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return os.Stdout
}

func (r Runner) stderr() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return os.Stderr
}

func (r Runner) environ() []string {
	if r.Environ != nil {
		return r.Environ()
	}
	return os.Environ()
}

func (r Runner) shell() string {
	if r.Shell != nil {
		return r.Shell()
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/zsh"
}

func (r Runner) look(file string, env []string) (string, error) {
	if r.Look != nil {
		return r.Look(file, env)
	}
	return lookPath(file, env)
}

func (r Runner) exec(argv0 string, argv []string, env []string) error {
	if r.Exec != nil {
		return r.Exec(argv0, argv, env)
	}
	return defaultExec(argv0, argv, env)
}

func writeVars(w io.Writer, name string, vars []profile.Var) {
	fmt.Fprintf(w, "cloud-shell profile: %s\n", name)
	for _, v := range vars {
		fmt.Fprintf(w, "%s=%s\n", v.Key, v.Value)
	}
}
