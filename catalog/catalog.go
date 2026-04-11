package catalog

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.json
var JSONBytes []byte

type Song struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

type Catalog struct {
	Songs []Song `json:"songs"`
}

var FullCatalog = mustParse()

func mustParse() Catalog {
	var c Catalog
	if err := json.Unmarshal(JSONBytes, &c); err != nil {
		panic(err)
	}
	return c
}
