package utils

import (
	"path/filepath"
)

func IsAllowed(p string, allowedRoots []string) bool {
	abs, _ := filepath.Abs(p)
	for _, r := range allowedRoots {
		if filepath.HasPrefix(abs, r) {
			return true
		}
	}
	return false
}
