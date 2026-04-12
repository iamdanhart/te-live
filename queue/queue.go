package queue

import (
	"sync"

	"github.com/iamdanhart/te-live/catalog"
)

// Entry is a single singer signup in the queue.
type Entry struct {
	Name  string
	Songs []catalog.Song
}

// Queue is a thread-safe in-memory list of singer signups.
type Queue struct {
	mu            sync.Mutex
	entries       []Entry
	SignupsOpen   bool
}

// New returns an empty Queue with signups open by default.
func New() *Queue {
	return &Queue{SignupsOpen: true}
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
	q.entries = append(q.entries, Entry{Name: name, Songs: songs})
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
