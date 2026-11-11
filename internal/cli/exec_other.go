//go:build !unix

package cli

import (
	"os"
	"os/exec"
)

func defaultExec(argv0 string, argv []string, env []string) error {
	cmd := exec.Command(argv0, argv[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
