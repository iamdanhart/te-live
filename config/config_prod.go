//go:build production

package config

import (
	"embed"
	"io"
)

//go:embed prod.json
var configFiles embed.FS

func openConfig(_ string) (io.ReadCloser, error) {
	return configFiles.Open("prod.json")
}