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

func LoadEnv() error {
	// Căutăm fișierul .env în mai multe locații posibile
	paths := []string{
		".env",
		"../.env",
		"../../.env",
	}

	var envFile *os.File
	var err error

	for _, path := range paths {
		envFile, err = os.Open(path)
		if err == nil {
			break
		}
	}

	// Dacă nu am găsit niciun fișier, încercăm directoarele executabilului
	if envFile == nil {
		exePath, execErr := os.Executable()
		if execErr == nil {
			envFile, err = os.Open(exePath + "/.env")
			if err != nil {
				envFile, err = os.Open(filepath.Dir(exePath) + "/.env")
			}
		}
	}

	if envFile == nil {
		return os.ErrNotExist
	}
	defer envFile.Close()

	// Citim fiecare linie din fișier
	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignorăm linii goale și comentarii
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Separăm la caracterul '='
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// Extragem cheia și valoarea
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Eliminăm ghilimele dacă există
		value = strings.Trim(value, "\"")

		// Setăm variabila de mediu
		os.Setenv(key, value)
	}

	return scanner.Err()
}
