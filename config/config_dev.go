//go:build !production

package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func openConfig(env string) (io.ReadCloser, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	return os.Open(filepath.Join(filepath.Dir(thisFile), fmt.Sprintf("%s.json", env)))
}