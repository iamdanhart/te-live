package queue

import (
	"sync"

	"github.com/iamdanhart/te-live/catalog"
)

// InMemQueue is a thread-safe in-memory list of singer signups.
type InMemQueue struct {
	mu          sync.Mutex
	entries     []Entry
	performed   []PerformedSong
	signupsOpen bool
}

// New returns an empty InMemQueue with signups open by default.
func NewInMemQueue() *InMemQueue {
	return &InMemQueue{signupsOpen: true}
}

func (q *InMemQueue) SignupsOpen() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.signupsOpen
}

// Entries returns a snapshot of all entries in the queue.
func (q *InMemQueue) Entries() []Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	snapshot := make([]Entry, len(q.entries))
	copy(snapshot, q.entries)
	return snapshot
}

// ToggleSignups flips the SignupsOpen flag and returns the new value.
func (q *InMemQueue) ToggleSignups() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.signupsOpen = !q.signupsOpen
	return q.signupsOpen
}

// Add appends a new entry to the end of the queue.
func (q *InMemQueue) Add(name string, songs []catalog.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	entries := make([]SongEntry, len(songs))
	for i, s := range songs {
		entries[i] = SongEntry{Song: s}
	}
	q.entries = append(q.entries, Entry{Name: name, Songs: entries})
}

// MarkSongPerformed sets the Performed flag on a matching song in the first entry.
func (q *InMemQueue) MarkSongPerformed(title, artist string) {
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
func (q *InMemQueue) RecordPerformed(singer string, song catalog.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.performed = append(q.performed, PerformedSong{Singer: singer, Song: song})
}

// Performed returns a snapshot of all songs that have been performed.
func (q *InMemQueue) Performed() []PerformedSong {
	q.mu.Lock()
	defer q.mu.Unlock()
	snapshot := make([]PerformedSong, len(q.performed))
	copy(snapshot, q.performed)
	return snapshot
}

// AddSongToFirst appends a song to the first entry in the queue.
func (q *InMemQueue) AddSongToFirst(song catalog.Song) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) == 0 {
		return
	}
	q.entries[0].Songs = append(q.entries[0].Songs, SongEntry{Song: song})
}

// MoveCurrentToBottom moves the first entry to the end of the queue.
func (q *InMemQueue) MoveCurrentToBottom() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) <= 1 {
		return
	}
	q.entries = append(q.entries[1:], q.entries[0])
}

// Current returns the first entry in the queue, or nil if empty.
func (q *InMemQueue) Current() *Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) == 0 {
		return nil
	}
	e := q.entries[0]
	return &e
}

// Next returns the second entry in the queue, or nil if fewer than two entries.
func (q *InMemQueue) Next() *Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) < 2 {
		return nil
	}
	e := q.entries[1]
	return &e
}
