package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wcampos/cloud-shell/internal/profile"
)

func testStore(t *testing.T) profile.Store {
	t.Helper()
	root := t.TempDir()
	return profile.Store{
		Root:      filepath.Join(root, "profiles"),
		AliasPath: filepath.Join(root, "aliases"),
	}
}

func runner(t *testing.T, store profile.Store) (Runner, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return Runner{
		Stdout: stdout,
		Stderr: stderr,
		Store:  store,
	}, stdout, stderr
}

func TestListMarksActive(t *testing.T) {
	store := testStore(t)
	if _, err := store.Init("hex"); err != nil {
		t.Fatalf("init hex: %v", err)
	}
	if _, err := store.Init("vse"); err != nil {
		t.Fatalf("init vse: %v", err)
	}
	t.Setenv("CLOUD_SHELL_PROFILE", "hex")
	t.Setenv("CLOUD_PROFILE", "")

	r, stdout, stderr := runner(t, store)
	if code := r.Run(nil); code != 0 {
		t.Fatalf("list code %d: %s", code, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "* hex\n") || !strings.Contains(got, "  vse\n") {
		t.Fatalf("list = %q", got)
	}
}

func TestListMissingRoot(t *testing.T) {
	r, _, stderr := runner(t, testStore(t))
	if code := r.Run([]string{"--list"}); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "No profiles yet") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEnvAndCurrent(t *testing.T) {
	store := testStore(t)
	lay, err := store.Init("hex")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.AliasPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(store.AliasPath, []byte("hexagon hex\n"), 0o644); err != nil {
		t.Fatalf("aliases: %v", err)
	}

	r, stdout, stderr := runner(t, store)
	if code := r.Run([]string{"env", "hexagon"}); code != 0 {
		t.Fatalf("env code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "export AZURE_CONFIG_DIR='"+lay.Azure+"'") {
		t.Fatalf("env = %s", stdout.String())
	}

	t.Setenv("CLOUD_SHELL_PROFILE", "")
	t.Setenv("CLOUD_PROFILE", "")
	t.Setenv("AZURE_CONFIG_DIR", "")
	stdout.Reset()
	if code := r.Run([]string{"current"}); code != 0 {
		t.Fatalf("current code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no profile active") {
		t.Fatalf("current = %s", stdout.String())
	}
}

func TestMissingProfileListsAvailable(t *testing.T) {
	store := testStore(t)
	if _, err := store.Init("viking"); err != nil {
		t.Fatalf("init: %v", err)
	}
	r, _, stderr := runner(t, store)
	if code := r.Run([]string{"env", "nope"}); code != 1 {
		t.Fatalf("code = %d", code)
	}
	got := stderr.String()
	if !strings.Contains(got, "not found") || !strings.Contains(got, "viking") {
		t.Fatalf("stderr = %s", got)
	}
}

func TestHookAndCompletion(t *testing.T) {
	r, stdout, stderr := runner(t, testStore(t))
	if code := r.Run([]string{"hook", "zsh"}); code != 0 {
		t.Fatalf("hook code %d: %s", code, stderr.String())
	}
	hook := stdout.String()
	if !strings.Contains(hook, "prompt_dir()") || !strings.Contains(hook, "profile.zsh") {
		t.Fatalf("zsh hook missing path label:\n%s", hook)
	}
	if !strings.Contains(hook, "shell/color") || !strings.Contains(hook, `prompt_segment "$(_cloud_shell_profile_color "$label")" black`) {
		t.Fatalf("zsh hook missing colored profile segment:\n%s", hook)
	}
	if !strings.Contains(hook, "CLOUD_PROFILE") {
		t.Fatal("zsh hook ignores CLOUD_PROFILE")
	}

	stdout.Reset()
	if code := r.Run([]string{"hook", "bash"}); code != 0 {
		t.Fatalf("bash hook code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "PS1=") || !strings.Contains(stdout.String(), "profile.bashrc") {
		t.Fatalf("bash hook = %s", stdout.String())
	}

	stdout.Reset()
	if code := r.Run([]string{"completion", "zsh"}); code != 0 {
		t.Fatalf("completion code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "#compdef cloud-shell") || !strings.Contains(stdout.String(), "complete --profiles") {
		t.Fatalf("zsh completion = %s", stdout.String())
	}

	if code := r.Run([]string{"completion", "fish"}); code != 1 {
		t.Fatalf("fish completion code = %d, want 1", code)
	}
}

func TestEnterDoesNotReplaceProcess(t *testing.T) {
	store := testStore(t)
	if _, err := store.Init("hex"); err != nil {
		t.Fatalf("init: %v", err)
	}
	var argv0 string
	var argv []string
	var env []string
	r, stdout, stderr := runner(t, store)
	r.Exec = func(got0 string, gotArgv []string, gotEnv []string) error {
		argv0 = got0
		argv = append([]string{}, gotArgv...)
		env = append([]string{}, gotEnv...)
		return nil
	}
	r.Shell = func() string { return "/bin/zsh" }
	r.Look = func(file string, _ []string) (string, error) {
		return file, nil
	}

	if code := r.Run([]string{"hex"}); code != 0 {
		t.Fatalf("enter code %d: %s", code, stderr.String())
	}
	if argv0 != "/bin/zsh" || strings.Join(argv, " ") != "/bin/zsh -l" {
		t.Fatalf("exec argv0=%s argv=%v", argv0, argv)
	}
	if !strings.Contains(stderr.String(), "entered profile 'hex'") {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("interactive enter wrote stdout: %s", stdout.String())
	}
	if !hasEnv(env, "CLOUD_SHELL_PROFILE", "hex") || !hasEnv(env, "KUBECONFIG", filepath.Join(store.Root, "hex", "kube", "config")) {
		t.Fatal("profile environment was not applied")
	}

	argv = nil
	if code := r.Run([]string{"hex", "echo", "hi"}); code != 0 {
		t.Fatalf("command code %d: %s", code, stderr.String())
	}
	if len(argv) < 2 || argv[0] != "echo" || argv[1] != "hi" {
		t.Fatalf("command argv = %v", argv)
	}
}

func TestVersion(t *testing.T) {
	r, stdout, stderr := runner(t, testStore(t))
	r.Version = "v1.2.3"
	if code := r.Run([]string{"version"}); code != 0 {
		t.Fatalf("version code %d: %s", code, stderr.String())
	}
	if stdout.String() != "cloud-shell v1.2.3\n" {
		t.Fatalf("version = %q", stdout.String())
	}

	stdout.Reset()
	r.Version = ""
	if code := r.Run([]string{"--version"}); code != 0 {
		t.Fatalf("--version code %d: %s", code, stderr.String())
	}
	if stdout.String() != "cloud-shell dev\n" {
		t.Fatalf("default version = %q", stdout.String())
	}
}

func TestUsage(t *testing.T) {
	r, _, stderr := runner(t, testStore(t))
	if code := r.Run([]string{"--bogus"}); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func hasEnv(env []string, key, value string) bool {
	want := key + "=" + value
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
