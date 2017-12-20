package file

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
)

func GetMysteriumDirectory(path string) string {
	dir, _ := homedir.Dir()

	dir = filepath.Join(dir, ".mysterium", path)

	return dir
}
