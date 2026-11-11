package profile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	return Store{
		Root:      filepath.Join(root, "profiles"),
		AliasPath: filepath.Join(root, "config", "aliases"),
	}
}

func writeAliases(t *testing.T, s Store, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(s.AliasPath), 0o755); err != nil {
		t.Fatalf("mkdir aliases: %v", err)
	}
	if err := os.WriteFile(s.AliasPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write aliases: %v", err)
	}
}

func TestCanonicalAlias(t *testing.T) {
	s := newStore(t)
	writeAliases(t, s, "# customers\nshort customer\nabbr customer\nalt label\nsame=same\n")

	got, err := s.Canonical("short")
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	if got != "customer" {
		t.Fatalf("Canonical(short) = %q, want customer", got)
	}

	got, err = s.Canonical("same")
	if err != nil {
		t.Fatalf("Canonical same: %v", err)
	}
	if got != "same" {
		t.Fatalf("Canonical(same) = %q, want same", got)
	}
}

func TestResolveMissing(t *testing.T) {
	s := newStore(t)
	_, err := s.Resolve("customer")
	var missing *NotFoundError
	if !errors.As(err, &missing) {
		t.Fatalf("Resolve err = %v, want NotFoundError", err)
	}
	if missing.Name != "customer" {
		t.Fatalf("Name = %q, want customer", missing.Name)
	}
	if _, statErr := os.Stat(filepath.Join(s.Root, "customer")); !os.IsNotExist(statErr) {
		t.Fatal("Resolve created a missing profile")
	}
}

func TestRejectBadNames(t *testing.T) {
	s := newStore(t)
	for _, name := range []string{"../etc", "..", "", "has space", "env", "init"} {
		if _, err := s.Canonical(name); err == nil {
			t.Fatalf("Canonical(%q) succeeded", name)
		}
	}
}

func TestInitKeepsCredentials(t *testing.T) {
	s := newStore(t)
	lay, err := s.Init("customer")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	secret := []byte("[default]\naws_access_key_id = secret\n")
	if err := os.WriteFile(lay.AWSCreds, secret, 0o600); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	if _, err := s.Init("customer"); err != nil {
		t.Fatalf("second Init: %v", err)
	}
	got, err := os.ReadFile(lay.AWSCreds)
	if err != nil {
		t.Fatalf("read credentials: %v", err)
	}
	if string(got) != string(secret) {
		t.Fatalf("credentials changed to %q", got)
	}

	info, err := os.Stat(lay.ZshSnippet)
	if err != nil {
		t.Fatalf("snippet: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("empty zsh snippet")
	}
	color, err := os.ReadFile(lay.Color)
	if err != nil {
		t.Fatalf("color: %v", err)
	}
	if string(color) != "magenta\n" {
		t.Fatalf("color = %q, want magenta", color)
	}
	for _, dir := range []string{lay.Azure, lay.GCloud, lay.Claude, lay.Dotnet, lay.NuGet} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}

func TestInitFollowsAlias(t *testing.T) {
	s := newStore(t)
	writeAliases(t, s, "short customer\n")
	lay, err := s.Init("short")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if lay.Name != "customer" {
		t.Fatalf("name = %q, want customer", lay.Name)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "short")); !os.IsNotExist(err) {
		t.Fatal("init created the alias name instead of the target")
	}
}

func TestExports(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my profiles")
	s := Store{
		Root:      root,
		AliasPath: filepath.Join(t.TempDir(), "aliases"),
	}
	lay, err := s.Init("customer")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	extra := "FOO=bar baz\n# comment\n\nCLOUDSDK_ACTIVE_CONFIG_NAME=customer\nQUOTE=\"it's\"\n"
	if err := os.WriteFile(lay.EnvFile, []byte(extra), 0o644); err != nil {
		t.Fatalf("env file: %v", err)
	}

	text, err := s.Exports("customer")
	if err != nil {
		t.Fatalf("Exports: %v", err)
	}
	checks := []string{
		"export CLOUD_SHELL_PROFILE='customer'",
		"export CLOUD_PROFILE='customer'",
		"export AZURE_CONFIG_DIR='" + lay.Azure + "'",
		"export AWS_CONFIG_FILE='" + lay.AWSConfig + "'",
		"export AWS_SHARED_CREDENTIALS_FILE='" + lay.AWSCreds + "'",
		"export CLOUDSDK_CONFIG='" + lay.GCloud + "'",
		"export KUBECONFIG='" + lay.KubeConfig + "'",
		"export CLAUDE_CONFIG_DIR='" + lay.Claude + "'",
		"export DOTNET_CLI_HOME='" + lay.Dotnet + "'",
		"export NUGET_PACKAGES='" + lay.NuGet + "'",
		"export FOO='bar baz'",
		"export CLOUDSDK_ACTIVE_CONFIG_NAME='customer'",
		`export QUOTE='it'\''s'`,
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Errorf("exports missing %s\n%s", want, text)
		}
	}
	if strings.Contains(text, "my profiles") && !strings.Contains(text, "'"+lay.Azure+"'") {
		t.Fatal("path with a space was not shell-quoted")
	}

	info, err := os.Stat(lay.AWSConfig)
	if err != nil {
		t.Fatalf("aws config: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("aws config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestBadEnvLine(t *testing.T) {
	s := newStore(t)
	lay, err := s.Init("customer")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(lay.EnvFile, []byte("NOT AN ASSIGNMENT\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	if _, err := s.Exports("customer"); err == nil {
		t.Fatal("expected error for malformed env file")
	}
}

func TestBadAliasLine(t *testing.T) {
	s := newStore(t)
	writeAliases(t, s, "only-one-field\n")
	if _, err := s.Canonical("customer"); err == nil {
		t.Fatal("expected alias parse error")
	}
}

func TestNames(t *testing.T) {
	s := newStore(t)
	writeAliases(t, s, "short customer\nabbr label\n")
	if _, err := s.Init("customer"); err != nil {
		t.Fatalf("Init customer: %v", err)
	}
	if _, err := s.Init("label"); err != nil {
		t.Fatalf("Init label: %v", err)
	}
	names, err := s.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	got := strings.Join(names, ",")
	if got != "abbr,customer,label,short" {
		t.Fatalf("Names = %q", got)
	}
}

func TestMergeEnv(t *testing.T) {
	vars := []Var{{Key: "A", Value: "new"}, {Key: "B", Value: "added"}}
	got := MergeEnv([]string{"A=old", "KEEP=1", "BARE"}, vars)
	want := []string{"A=new", "KEEP=1", "BARE", "B=added"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("MergeEnv = %#v, want %#v", got, want)
	}
}

func TestDescribeDoesNotCreate(t *testing.T) {
	s := newStore(t)
	if err := os.MkdirAll(filepath.Join(s.Root, "customer"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := s.Describe("customer"); err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "customer", "aws", "config")); !os.IsNotExist(err) {
		t.Fatal("Describe created aws config")
	}
}
