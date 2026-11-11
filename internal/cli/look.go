package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func lookPath(file string, env []string) (string, error) {
	if file == "" {
		return "", errors.New("empty command")
	}
	if strings.ContainsRune(file, os.PathSeparator) {
		info, err := os.Stat(file)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory", file)
		}
		return file, nil
	}
	pathEnv, ok := pathValue(env)
	if !ok || pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	return lookInPath(file, pathEnv)
}

func pathValue(env []string) (string, bool) {
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if key == "PATH" || (runtime.GOOS == "windows" && strings.EqualFold(key, "path")) {
			return value, true
		}
	}
	return "", false
}

func lookInPath(file, pathEnv string) (resolved string, err error) {
	previous, had := os.LookupEnv("PATH")
	if err = os.Setenv("PATH", pathEnv); err != nil {
		return "", fmt.Errorf("setting PATH: %w", err)
	}
	defer func() {
		var restoreErr error
		if had {
			restoreErr = os.Setenv("PATH", previous)
		} else {
			restoreErr = os.Unsetenv("PATH")
		}
		if err == nil && restoreErr != nil {
			err = fmt.Errorf("restoring PATH: %w", restoreErr)
		}
	}()
	resolved, err = exec.LookPath(file)
	if err != nil {
		return "", fmt.Errorf("executable %q not found in PATH", file)
	}
	return resolved, nil
}
