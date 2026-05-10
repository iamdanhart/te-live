package router

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/iamdanhart/te-live/queue"
	"github.com/stretchr/testify/assert"
)

// stubQueue implements queue.Queue for testing queueStatusData.
// Only Entries and SignupsOpen are implemented; all other methods panic.
type stubQueue struct {
	entries     []queue.Entry
	signupsOpen bool
}

func (s *stubQueue) Entries() []queue.Entry    { return s.entries }
func (s *stubQueue) SignupsOpen() bool          { return s.signupsOpen }
func (s *stubQueue) Songs() []queue.Song        { panic("not implemented") }
func (s *stubQueue) ToggleSignups() bool        { panic("not implemented") }
func (s *stubQueue) Add(string, []int) error    { panic("not implemented") }
func (s *stubQueue) CompleteCurrentSong(string, int) { panic("not implemented") }
func (s *stubQueue) Performed() []queue.PerformedSong { panic("not implemented") }
func (s *stubQueue) AddSongToFirst(int)         { panic("not implemented") }
func (s *stubQueue) MoveCurrentToBottom()       { panic("not implemented") }
func (s *stubQueue) RemoveCurrent()             { panic("not implemented") }
func (s *stubQueue) MoveEntry(int, int)         { panic("not implemented") }
func (s *stubQueue) HasName(string) bool              { panic("not implemented") }
func (s *stubQueue) AuthenticateHost(string) bool     { panic("not implemented") }

func TestHandleSignup_EmptyName(t *testing.T) {
	form := url.Values{"name": {""}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleSignup(rr, req, nil)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestHandleSignup_WhitespaceName(t *testing.T) {
	form := url.Values{"name": {"   "}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleSignup(rr, req, nil)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestHandleSignup_NoSongs(t *testing.T) {
	form := url.Values{"name": {"Dan"}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleSignup(rr, req, nil)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "at least one song is required")
}

func TestHandleSignup_NameTooLong(t *testing.T) {
	form := url.Values{"name": {strings.Repeat("a", 51)}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleSignup(rr, req, nil)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name too long")
}

func TestQueueStatusData_EmptyQueue(t *testing.T) {
	q := &stubQueue{entries: nil, signupsOpen: false}
	data := queueStatusData(q)
	assert.Nil(t, data.Current)
	assert.Nil(t, data.Next)
	assert.False(t, data.SignupsOpen)
}

func TestQueueStatusData_OneEntry(t *testing.T) {
	entries := []queue.Entry{{ID: 1, Name: "Alice"}}
	q := &stubQueue{entries: entries, signupsOpen: true}
	data := queueStatusData(q)
	assert.Equal(t, "Alice", data.Current.Name)
	assert.Nil(t, data.Next)
	assert.True(t, data.SignupsOpen)
}

func TestQueueStatusData_TwoEntries(t *testing.T) {
	entries := []queue.Entry{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	q := &stubQueue{entries: entries, signupsOpen: false}
	data := queueStatusData(q)
	assert.Equal(t, "Alice", data.Current.Name)
	assert.Equal(t, "Bob", data.Next.Name)
}

func TestQueueStatusData_ManyEntries(t *testing.T) {
	entries := []queue.Entry{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Carol"},
	}
	q := &stubQueue{entries: entries, signupsOpen: true}
	data := queueStatusData(q)
	assert.Equal(t, "Alice", data.Current.Name)
	// Next is always only the second entry, regardless of queue length
	assert.Equal(t, "Bob", data.Next.Name)
}