package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Clean cleans a path by removing surrounding quotes and trailing separators.
func Clean(dir string) string {
	dir = strings.Trim(dir, `"`)
	dir = filepath.Clean(dir)
	dir = strings.TrimRight(dir, `\/`)
	return dir
}

// LoadEnv finds a .env file and sets its keys as process environment
// variables. It searches the current directory, then two levels of parents,
// then the executable's directory. Variables already set in the environment
// are left untouched, so real env vars always win over the .env file.
//
// It returns os.ErrNotExist if no .env file was found in any location.
func LoadEnv() error {
	candidates := []string{".env", "../.env", "../../.env"}
	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), ".env"))
	}

	var envFile *os.File
	for _, path := range candidates {
		f, err := os.Open(path)
		if err == nil {
			envFile = f
			break
		}
	}
	if envFile == nil {
		return os.ErrNotExist
	}
	defer envFile.Close()

	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}
