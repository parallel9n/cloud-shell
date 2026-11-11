package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	pathEnv := "/usr/bin:/bin"
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok && key == "PATH" {
			pathEnv = value
			break
		}
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, file)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("executable %q not found in PATH", file)
}
