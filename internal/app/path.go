package app

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandHomePath(input string) string {
	if input == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(input, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(input, "~/"))
		}
	}
	return input
}

func ResolveOutputDir(cwd, input string) (string, error) {
	expanded := ExpandHomePath(input)
	candidate := expanded
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(cwd, candidate)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errNotDir(resolved)
	}
	return filepath.Abs(resolved)
}

type errNotDir string

func (e errNotDir) Error() string {
	return string(e) + " is not a directory"
}
