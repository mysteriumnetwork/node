package cmd

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
)

// GetMysteriumDirectory makes path to full path in home directory
func GetMysteriumDirectory(path string) string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, ".mysterium", path)
}
