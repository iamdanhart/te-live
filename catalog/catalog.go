package catalog

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.json
var JSONBytes []byte

type Song struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	TabUrl string `json:"tabUrl,omitempty"`
}

type Catalog struct {
	Songs []Song `json:"songs"`
}

var FullCatalog = mustParse()
var byID = buildIDIndex()

func buildIDIndex() map[int]Song {
	m := make(map[int]Song, len(FullCatalog.Songs))
	for _, s := range FullCatalog.Songs {
		m[s.ID] = s
	}
	return m
}

func FindByID(id int) (Song, bool) {
	s, ok := byID[id]
	return s, ok
}

func mustParse() Catalog {
	var c Catalog
	if err := json.Unmarshal(JSONBytes, &c); err != nil {
		panic(err)
	}
	return c
}
