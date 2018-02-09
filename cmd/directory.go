package cmd

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
)

// GetDataDirectory makes full path to application's data
func GetDataDirectory() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, ".mysterium")
}
