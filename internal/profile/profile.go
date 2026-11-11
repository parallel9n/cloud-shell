// Package profile maps a customer name to isolated cloud CLI config directories.
package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	dirMode  = 0o755
	fileMode = 0o644
	credMode = 0o600
)

const (
	envProfile    = "CLOUD_SHELL_PROFILE"
	envProfileAlt = "CLOUD_PROFILE"
	envAzure      = "AZURE_CONFIG_DIR"
	envAWSConfig  = "AWS_CONFIG_FILE"
	envAWSCreds   = "AWS_SHARED_CREDENTIALS_FILE"
	envGCloud     = "CLOUDSDK_CONFIG"
	envKube       = "KUBECONFIG"
	envClaude     = "CLAUDE_CONFIG_DIR"
	envDotnet     = "DOTNET_CLI_HOME"
	envNuGet      = "NUGET_PACKAGES"
)

// standardKeys is the export order for managed variables.
var standardKeys = []string{
	envProfile,
	envProfileAlt,
	envAzure,
	envAWSConfig,
	envAWSCreds,
	envGCloud,
	envKube,
	envClaude,
	envDotnet,
	envNuGet,
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// reserved names are CLI subcommands, so they cannot be profile directories.
var reserved = map[string]struct{}{
	"complete":   {},
	"completion": {},
	"current":    {},
	"env":        {},
	"help":       {},
	"hook":       {},
	"init":       {},
	"list":       {},
}

const zshSnippet = `# Optional commands for this cloud-shell profile.
# Sourced when this profile is active. ~/.zshrc still loads Oh My Zsh.
`

const bashSnippet = `# Optional commands for this cloud-shell profile.
# Sourced when this profile is active.
`

const envHeader = `# Extra KEY=value pairs for this profile.
# Blank lines and comments are ignored. Values may be quoted.
`

// colorDefault is the Agnoster background used when shell/color is absent.
const colorDefault = "magenta\n"

// NotFoundError means the named profile directory does not exist.
type NotFoundError struct {
	Name string
	Path string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("profile %q not found at %s", e.Name, e.Path)
}

// Var is one environment variable exported for a profile.
type Var struct {
	Key   string
	Value string
}

// Layout is the on-disk tree for one profile.
type Layout struct {
	Name        string
	Root        string
	Azure       string
	AWSConfig   string
	AWSCreds    string
	GCloud      string
	KubeConfig  string
	Claude      string
	Dotnet      string
	NuGet       string
	ZshSnippet  string
	BashSnippet string
	Color       string
	EnvFile     string
}

// Store locates profile directories and the alias file.
// Root and AliasPath must be set. Default fills both from the home directory.
type Store struct {
	Root      string
	AliasPath string
}

// Default returns a Store rooted at CLOUD_PROFILES_ROOT or ~/.cloud-profiles.
func Default() (Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Store{}, fmt.Errorf("home directory: %w", err)
	}
	root := os.Getenv("CLOUD_PROFILES_ROOT")
	if root == "" {
		root = filepath.Join(home, ".cloud-profiles")
	}
	aliasPath := os.Getenv("CLOUD_SHELL_ALIAS_FILE")
	if aliasPath == "" {
		aliasPath = filepath.Join(home, ".config", "cloud-shell", "aliases")
	}
	return Store{
		Root:      root,
		AliasPath: aliasPath,
	}, nil
}

// Active returns the profile selected in the environment, if any.
func Active() string {
	if name := os.Getenv(envProfile); name != "" {
		return name
	}
	return os.Getenv(envProfileAlt)
}

// Canonical maps an alias to a directory name. The directory does not need to exist.
func (s Store) Canonical(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	aliases, err := s.aliases()
	if err != nil {
		return "", err
	}
	if target, ok := aliases[name]; ok {
		name = target
	}
	if err := validateName(name); err != nil {
		return "", err
	}
	return name, nil
}

// Resolve maps an alias to an existing profile directory name.
func (s Store) Resolve(name string) (string, error) {
	canonical, err := s.Canonical(name)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.Root, canonical)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &NotFoundError{Name: canonical, Path: path}
		}
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("profile %q is not a directory (%s)", canonical, path)
	}
	return canonical, nil
}

// List returns profile directory names. A missing root is os.ErrNotExist.
func (s Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return nil, fmt.Errorf("reading profiles in %s: %w", s.Root, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// Names returns profile directories and alias names for completion.
func (s Store) Names() ([]string, error) {
	dirs, err := s.List()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	aliases, err := s.aliases()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(dirs)+len(aliases))
	for _, name := range dirs {
		seen[name] = struct{}{}
	}
	for alias := range aliases {
		seen[alias] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// Init creates the profile tree and starter snippets. Existing files are left in place.
func (s Store) Init(name string) (Layout, error) {
	canonical, err := s.Canonical(name)
	if err != nil {
		return Layout{}, err
	}
	lay := s.layout(canonical)
	if err := os.MkdirAll(lay.Root, dirMode); err != nil {
		return Layout{}, fmt.Errorf("creating %s: %w", lay.Root, err)
	}
	if err := ensureTree(lay); err != nil {
		return Layout{}, err
	}
	files := []struct {
		path string
		data []byte
		mode os.FileMode
	}{
		{path: lay.ZshSnippet, data: []byte(zshSnippet), mode: fileMode},
		{path: lay.BashSnippet, data: []byte(bashSnippet), mode: fileMode},
		{path: lay.Color, data: []byte(colorDefault), mode: fileMode},
		{path: lay.EnvFile, data: []byte(envHeader), mode: fileMode},
	}
	for _, file := range files {
		if err := writeNew(file.path, file.data, file.mode); err != nil {
			return Layout{}, err
		}
	}
	return lay, nil
}

// Vars returns the environment for an existing profile, creating missing config dirs.
func (s Store) Vars(name string) ([]Var, error) {
	canonical, err := s.Resolve(name)
	if err != nil {
		return nil, err
	}
	lay := s.layout(canonical)
	if err := ensureTree(lay); err != nil {
		return nil, err
	}
	return bindVars(lay)
}

// Describe returns the environment for an existing profile without creating files.
func (s Store) Describe(name string) ([]Var, error) {
	canonical, err := s.Resolve(name)
	if err != nil {
		return nil, err
	}
	return bindVars(s.layout(canonical))
}

// Exports returns POSIX export lines for an existing profile.
func (s Store) Exports(name string) (string, error) {
	vars, err := s.Vars(name)
	if err != nil {
		return "", err
	}
	return FormatExports(vars), nil
}

// FormatExports renders vars as POSIX export lines.
func FormatExports(vars []Var) string {
	var b strings.Builder
	for _, v := range vars {
		fmt.Fprintf(&b, "export %s=%s\n", v.Key, shellQuote(v.Value))
	}
	return b.String()
}

// MergeEnv overlays vars onto base. Later profile keys replace earlier values.
func MergeEnv(base []string, vars []Var) []string {
	overlay := make(map[string]string, len(vars))
	order := make([]string, len(vars))
	for i, v := range vars {
		overlay[v.Key] = v.Value
		order[i] = v.Key
	}
	out := make([]string, 0, len(base)+len(vars))
	seen := make(map[string]struct{}, len(vars))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			out = append(out, item)
			continue
		}
		if value, replaced := overlay[key]; replaced {
			out = append(out, key+"="+value)
			seen[key] = struct{}{}
			continue
		}
		out = append(out, item)
	}
	for _, key := range order {
		if _, ok := seen[key]; ok {
			continue
		}
		out = append(out, key+"="+overlay[key])
	}
	return out
}

func (s Store) layout(name string) Layout {
	root := filepath.Join(s.Root, name)
	dotnet := filepath.Join(root, "dotnet")
	return Layout{
		Name:        name,
		Root:        root,
		Azure:       filepath.Join(root, "azure"),
		AWSConfig:   filepath.Join(root, "aws", "config"),
		AWSCreds:    filepath.Join(root, "aws", "credentials"),
		GCloud:      filepath.Join(root, "gcloud"),
		KubeConfig:  filepath.Join(root, "kube", "config"),
		Claude:      filepath.Join(root, "claude"),
		Dotnet:      dotnet,
		NuGet:       filepath.Join(dotnet, "nuget", "packages"),
		ZshSnippet:  filepath.Join(root, "shell", "profile.zsh"),
		BashSnippet: filepath.Join(root, "shell", "profile.bashrc"),
		Color:       filepath.Join(root, "shell", "color"),
		EnvFile:     filepath.Join(root, "env"),
	}
}

func (s Store) aliases() (map[string]string, error) {
	data, err := os.ReadFile(s.AliasPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading aliases %s: %w", s.AliasPath, err)
	}
	parsed, err := parseAliases(data)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func ensureTree(lay Layout) error {
	dirs := []string{
		lay.Azure,
		filepath.Dir(lay.AWSConfig),
		lay.GCloud,
		filepath.Dir(lay.KubeConfig),
		lay.Claude,
		lay.NuGet,
		filepath.Dir(lay.ZshSnippet),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	for _, path := range []string{lay.AWSConfig, lay.AWSCreds} {
		if err := writeNew(path, nil, credMode); err != nil {
			return err
		}
	}
	return nil
}

func bindVars(lay Layout) ([]Var, error) {
	extra, err := readEnvFile(lay.EnvFile)
	if err != nil {
		return nil, err
	}
	vals := map[string]string{
		envProfile:    lay.Name,
		envProfileAlt: lay.Name,
		envAzure:      lay.Azure,
		envAWSConfig:  lay.AWSConfig,
		envAWSCreds:   lay.AWSCreds,
		envGCloud:     lay.GCloud,
		envKube:       lay.KubeConfig,
		envClaude:     lay.Claude,
		envDotnet:     lay.Dotnet,
		envNuGet:      lay.NuGet,
	}
	for key, value := range extra {
		vals[key] = value
	}
	seen := make(map[string]struct{}, len(standardKeys))
	for _, key := range standardKeys {
		seen[key] = struct{}{}
	}
	rest := make([]string, 0)
	for key := range vals {
		if _, ok := seen[key]; ok {
			continue
		}
		rest = append(rest, key)
	}
	sort.Strings(rest)
	keys := append(append([]string{}, standardKeys...), rest...)
	vars := make([]Var, 0, len(keys))
	for _, key := range keys {
		vars = append(vars, Var{Key: key, Value: vals[key]})
	}
	return vars, nil
}

func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	parsed, err := parseEnv(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return parsed, nil
}

func parseEnv(data []byte) (map[string]string, error) {
	vals := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: want KEY=value", i+1)
		}
		key = strings.TrimSpace(key)
		if !envKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("line %d: invalid key %q", i+1, key)
		}
		value, err := parseValue(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		vals[key] = value
	}
	return vals, nil
}

func parseValue(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	switch raw[0] {
	case '"':
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("bad quoted value: %w", err)
		}
		return value, nil
	case '\'':
		if len(raw) < 2 || raw[len(raw)-1] != '\'' {
			return "", errors.New("unterminated single quote")
		}
		return raw[1 : len(raw)-1], nil
	default:
		return raw, nil
	}
}

func parseAliases(data []byte) (map[string]string, error) {
	aliases := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		alias, target, err := splitAlias(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if err := validateName(alias); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if err := validateName(target); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		aliases[alias] = target
	}
	return aliases, nil
}

func splitAlias(line string) (string, string, error) {
	if alias, target, ok := strings.Cut(line, "="); ok {
		alias = strings.TrimSpace(alias)
		target = strings.TrimSpace(target)
		if alias == "" || target == "" {
			return "", "", errors.New("want alias and profile")
		}
		return alias, target, nil
	}
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return "", "", errors.New("want alias and profile")
	}
	return fields[0], fields[1], nil
}

func validateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q", name)
	}
	if _, ok := reserved[name]; ok {
		return fmt.Errorf("profile name %q is reserved", name)
	}
	return nil
}

func writeNew(path string, data []byte, mode os.FileMode) (err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer func() {
		cerr := f.Close()
		if err == nil && cerr != nil {
			err = fmt.Errorf("closing %s: %w", path, cerr)
		}
	}()
	if len(data) == 0 {
		return nil
	}
	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
