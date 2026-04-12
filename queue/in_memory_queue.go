package queue

import (
	"sync"

	"github.com/iamdanhart/te-live/catalog"
)

// SongEntry pairs a song with a flag indicating whether it has been performed.
type SongEntry struct {
	Song      catalog.Song
	Performed bool
}

// Entry is a single singer signup in the queue.
type Entry struct {
	Name  string
	Songs []SongEntry
}

// PerformedSong records a song that has been sung, along with the singer's name.
type PerformedSong struct {
	Singer string
	Song   catalog.Song
}

// Queue is a thread-safe in-memory list of singer signups.
type Queue struct {
	mu          sync.Mutex
	entries     []Entry
	performed   []PerformedSong
	SignupsOpen bool
}

// New returns an empty Queue with signups open by default.
func New() *Queue {
	return &Queue{SignupsOpen: true}
}

// Entries returns a snapshot of all entries in the queue.
func (q *Queue) Entries() []Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	snapshot := make([]Entry, len(q.entries))
	copy(snapshot, q.entries)
	return snapshot
}

// ToggleSignups flips the SignupsOpen flag and returns the new value.
func (q *Queue) ToggleSignups() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.SignupsOpen = !q.SignupsOpen
	return q.SignupsOpen
}

// Add appends a new entry to the end of the queue.
func (q *Queue) Add(name string, songs []catalog.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	entries := make([]SongEntry, len(songs))
	for i, s := range songs {
		entries[i] = SongEntry{Song: s}
	}
	q.entries = append(q.entries, Entry{Name: name, Songs: entries})
}

// MarkSongPerformed sets the Performed flag on a matching song in the first entry.
func (q *Queue) MarkSongPerformed(title, artist string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) == 0 {
		return
	}
	for i, s := range q.entries[0].Songs {
		if s.Song.Title == title && s.Song.Artist == artist {
			q.entries[0].Songs[i].Performed = true
			return
		}
	}
}

// RecordPerformed appends a song to the performed history.
func (q *Queue) RecordPerformed(singer string, song catalog.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.performed = append(q.performed, PerformedSong{Singer: singer, Song: song})
}

// Performed returns a snapshot of all songs that have been performed.
func (q *Queue) Performed() []PerformedSong {
	q.mu.Lock()
	defer q.mu.Unlock()
	snapshot := make([]PerformedSong, len(q.performed))
	copy(snapshot, q.performed)
	return snapshot
}

// AddSongToFirst appends a song to the first entry in the queue.
func (q *Queue) AddSongToFirst(song catalog.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) == 0 {
		return
	}
	q.entries[0].Songs = append(q.entries[0].Songs, SongEntry{Song: song})
}

// MoveCurrentToBottom moves the first entry to the end of the queue.
func (q *Queue) MoveCurrentToBottom() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) <= 1 {
		return
	}
	q.entries = append(q.entries[1:], q.entries[0])
}

// Current returns the first entry in the queue, or nil if empty.
func (q *Queue) Current() *Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) == 0 {
		return nil
	}
	e := q.entries[0]
	return &e
}

// Next returns the second entry in the queue, or nil if fewer than two entries.
func (q *Queue) Next() *Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) < 2 {
		return nil
	}
	e := q.entries[1]
	return &e
}
