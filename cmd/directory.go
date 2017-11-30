package cmd

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
)

func GetDirectory(paths ...string) string {
	dir, _ := homedir.Dir()

	dir = filepath.Join(dir, ".mysterium", filepath.Join(paths...))

	return dir
}
