package queue

import "context"

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
	ID           int
	Name         string
	Songs        []SongEntry
	TimesOnStage int
}

// PerformedSong records a song that has been sung, along with the singer's name.
type PerformedSong struct {
	Singer string
	Song   Song
}

type Queue interface {
	Entries(ctx context.Context) []Entry
	Songs(ctx context.Context) []Song
	SignupsOpen(ctx context.Context) bool
	ToggleSignups(ctx context.Context) (bool, error)
	Add(ctx context.Context, name string, songIDs []int) error
	CompleteCurrentSong(ctx context.Context, singer string, songID int) error
	Performed(ctx context.Context) []PerformedSong
	AddSongToFirst(ctx context.Context, songID int) error
	MoveCurrentToBottom(ctx context.Context) error
	RemoveCurrent(ctx context.Context) error
	MoveEntry(ctx context.Context, id, afterID int) error
	HasName(ctx context.Context, name string) bool
	AuthenticateHost(ctx context.Context, passcode string) bool
}
