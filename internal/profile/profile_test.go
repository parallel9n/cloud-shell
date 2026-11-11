package profile

import (
	"errors"
	"os"
	"path/filepath"
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
	writeAliases(t, s, "# customers\nhexagon hex\nhxr hex\nvik viking\nvse=vse\n")

	got, err := s.Canonical("hexagon")
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	if got != "hex" {
		t.Fatalf("Canonical(hexagon) = %q, want hex", got)
	}

	got, err = s.Canonical("vse")
	if err != nil {
		t.Fatalf("Canonical vse: %v", err)
	}
	if got != "vse" {
		t.Fatalf("Canonical(vse) = %q, want vse", got)
	}
}

func TestResolveMissing(t *testing.T) {
	s := newStore(t)
	_, err := s.Resolve("hex")
	var missing *NotFoundError
	if !errors.As(err, &missing) {
		t.Fatalf("Resolve err = %v, want NotFoundError", err)
	}
	if missing.Name != "hex" {
		t.Fatalf("Name = %q, want hex", missing.Name)
	}
	if _, statErr := os.Stat(filepath.Join(s.Root, "hex")); !os.IsNotExist(statErr) {
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
	lay, err := s.Init("hex")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	secret := []byte("[default]\naws_access_key_id = secret\n")
	if err := os.WriteFile(lay.AWSCreds, secret, 0o600); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	if _, err := s.Init("hex"); err != nil {
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
	writeAliases(t, s, "hexagon hex\n")
	lay, err := s.Init("hexagon")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if lay.Name != "hex" {
		t.Fatalf("name = %q, want hex", lay.Name)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "hexagon")); !os.IsNotExist(err) {
		t.Fatal("init created the alias name instead of the target")
	}
}

func TestExports(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my profiles")
	s := Store{
		Root:      root,
		AliasPath: filepath.Join(t.TempDir(), "aliases"),
	}
	lay, err := s.Init("hex")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	extra := "FOO=bar baz\n# comment\n\nCLOUDSDK_ACTIVE_CONFIG_NAME=hex\nQUOTE=\"it's\"\n"
	if err := os.WriteFile(lay.EnvFile, []byte(extra), 0o644); err != nil {
		t.Fatalf("env file: %v", err)
	}

	text, err := s.Exports("hex")
	if err != nil {
		t.Fatalf("Exports: %v", err)
	}
	checks := []string{
		"export CLOUD_SHELL_PROFILE='hex'",
		"export CLOUD_PROFILE='hex'",
		"export AZURE_CONFIG_DIR='" + lay.Azure + "'",
		"export AWS_CONFIG_FILE='" + lay.AWSConfig + "'",
		"export AWS_SHARED_CREDENTIALS_FILE='" + lay.AWSCreds + "'",
		"export CLOUDSDK_CONFIG='" + lay.GCloud + "'",
		"export KUBECONFIG='" + lay.KubeConfig + "'",
		"export CLAUDE_CONFIG_DIR='" + lay.Claude + "'",
		"export DOTNET_CLI_HOME='" + lay.Dotnet + "'",
		"export NUGET_PACKAGES='" + lay.NuGet + "'",
		"export FOO='bar baz'",
		"export CLOUDSDK_ACTIVE_CONFIG_NAME='hex'",
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
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("aws config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestBadEnvLine(t *testing.T) {
	s := newStore(t)
	lay, err := s.Init("hex")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(lay.EnvFile, []byte("NOT AN ASSIGNMENT\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}
	if _, err := s.Exports("hex"); err == nil {
		t.Fatal("expected error for malformed env file")
	}
}

func TestBadAliasLine(t *testing.T) {
	s := newStore(t)
	writeAliases(t, s, "only-one-field\n")
	if _, err := s.Canonical("hex"); err == nil {
		t.Fatal("expected alias parse error")
	}
}

func TestNames(t *testing.T) {
	s := newStore(t)
	writeAliases(t, s, "hexagon hex\nvik viking\n")
	if _, err := s.Init("hex"); err != nil {
		t.Fatalf("Init hex: %v", err)
	}
	if _, err := s.Init("viking"); err != nil {
		t.Fatalf("Init viking: %v", err)
	}
	names, err := s.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	got := strings.Join(names, ",")
	if got != "hex,hexagon,vik,viking" {
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
	if err := os.MkdirAll(filepath.Join(s.Root, "hex"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := s.Describe("hex"); err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "hex", "aws", "config")); !os.IsNotExist(err) {
		t.Fatal("Describe created aws config")
	}
}
