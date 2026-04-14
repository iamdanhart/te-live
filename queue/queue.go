package queue

// Song is a single song in the catalog.
type Song struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	TabUrl string `json:"tabUrl,omitempty"`
}

// SongEntry pairs a song with a flag indicating whether it has been performed.
type SongEntry struct {
	Song      Song
	Performed bool
}

// Entry is a single singer signup in the queue.
type Entry struct {
	ID    int
	Name  string
	Songs []SongEntry
}

// PerformedSong records a song that has been sung, along with the singer's name.
type PerformedSong struct {
	Singer string
	Song   Song
}

type Queue interface {
	Entries() []Entry
	Songs() []Song
	SignupsOpen() bool
	ToggleSignups() bool
	Add(name string, songIDs []int) error
	CompleteCurrentSong(singer string, songID int)
	Performed() []PerformedSong
	AddSongToFirst(songID int)
	MoveCurrentToBottom()
	RemoveCurrent()
	MoveEntry(id, afterID int)
	HasName(name string) bool
}
