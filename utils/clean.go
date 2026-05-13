package utils

import (
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
