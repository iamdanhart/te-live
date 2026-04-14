package catalog

type Song struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	TabUrl string `json:"tabUrl,omitempty"`
}