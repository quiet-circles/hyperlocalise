package envloader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/subosito/gotenv"
)

var filenames = []string{".env", ".env.local"}

// LoadProjectFiles loads .env and .env.local from the current working directory.
func LoadProjectFiles() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory for env loading: %w", err)
	}

	return LoadFilesInDir(cwd)
}

// LoadFilesInDir loads .env files from dir without overriding existing process environment variables.
func LoadFilesInDir(dir string) error {
	initial := currentEnvKeys()
	values := map[string]string{}

	for _, name := range filenames {
		path := filepath.Join(dir, name)
		env, err := readEnvFile(path)
		if err != nil {
			return err
		}

		for key, value := range env {
			values[key] = value
		}
	}

	for key, value := range values {
		if _, exists := initial[key]; exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s from env files: %w", key, err)
		}
	}

	return nil
}

func readEnvFile(path string) (map[string]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read env file metadata %s: %w", path, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("env path %s must be a file", path)
	}

	env, err := gotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("parse env file %s: %w", path, err)
	}

	return env, nil
}

func currentEnvKeys() map[string]struct{} {
	keys := make(map[string]struct{}, len(os.Environ()))

	for _, pair := range os.Environ() {
		if idx := strings.IndexByte(pair, '='); idx > 0 {
			keys[pair[:idx]] = struct{}{}
		}
	}

	return keys
}
