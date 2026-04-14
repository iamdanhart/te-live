package queue

import "github.com/iamdanhart/te-live/catalog"

// SongEntry pairs a song with a flag indicating whether it has been performed.
type SongEntry struct {
	Song      catalog.Song
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
	Song   catalog.Song
}

type Queue interface {
	Entries() []Entry
	SignupsOpen() bool
	ToggleSignups() bool
	Add(name string, songs []catalog.Song) error
	CompleteCurrentSong(singer string, song catalog.Song)
	Performed() []PerformedSong
	AddSongToFirst(song catalog.Song)
	MoveCurrentToBottom()
	RemoveCurrent()
	MoveEntry(id, afterID int)
	HasName(name string) bool
}
