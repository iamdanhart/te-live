package router

import (
	"context"
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

func (s *stubQueue) Entries(context.Context) []queue.Entry    { return s.entries }
func (s *stubQueue) SignupsOpen(context.Context) bool          { return s.signupsOpen }
func (s *stubQueue) Songs(context.Context) []queue.Song        { panic("not implemented") }
func (s *stubQueue) ToggleSignups(context.Context) bool        { panic("not implemented") }
func (s *stubQueue) Add(context.Context, string, []int) error  { panic("not implemented") }
func (s *stubQueue) CompleteCurrentSong(context.Context, string, int) error { panic("not implemented") }
func (s *stubQueue) Performed(context.Context) []queue.PerformedSong         { panic("not implemented") }
func (s *stubQueue) AddSongToFirst(context.Context, int) error               { panic("not implemented") }
func (s *stubQueue) MoveCurrentToBottom(context.Context) error               { panic("not implemented") }
func (s *stubQueue) RemoveCurrent(context.Context) error                     { panic("not implemented") }
func (s *stubQueue) MoveEntry(context.Context, int, int) error               { panic("not implemented") }
func (s *stubQueue) HasName(context.Context, string) bool             { panic("not implemented") }
func (s *stubQueue) AuthenticateHost(context.Context, string) bool    { panic("not implemented") }

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
	data := queueStatusData(context.Background(), q)
	assert.Nil(t, data.Current)
	assert.Nil(t, data.Next)
	assert.False(t, data.SignupsOpen)
}

func TestQueueStatusData_OneEntry(t *testing.T) {
	entries := []queue.Entry{{ID: 1, Name: "Alice"}}
	q := &stubQueue{entries: entries, signupsOpen: true}
	data := queueStatusData(context.Background(), q)
	assert.Equal(t, "Alice", data.Current.Name)
	assert.Nil(t, data.Next)
	assert.True(t, data.SignupsOpen)
}

func TestQueueStatusData_TwoEntries(t *testing.T) {
	entries := []queue.Entry{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	q := &stubQueue{entries: entries, signupsOpen: false}
	data := queueStatusData(context.Background(), q)
	assert.Equal(t, "Alice", data.Current.Name)
	assert.Equal(t, "Bob", data.Next.Name)
}

// addableStub extends stubQueue with a working Add method for signup success tests.
type addableStub struct {
	*stubQueue
	addErr error
}

func (s *addableStub) Add(_ context.Context, _ string, _ []int) error { return s.addErr }

// songsStub extends stubQueue with a working Songs method for catalog tests.
type songsStub struct {
	*stubQueue
	songs []queue.Song
}

func (s *songsStub) Songs(_ context.Context) []queue.Song { return s.songs }

func TestHandleSignup_Success(t *testing.T) {
	form := url.Values{"name": {"Alice"}, "song": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	q := &addableStub{stubQueue: &stubQueue{}}
	handleSignup(rr, req, q)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Alice")
}

func TestHandleSignup_AddError(t *testing.T) {
	form := url.Values{"name": {"Alice"}, "song": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	q := &addableStub{stubQueue: &stubQueue{}, addErr: queue.ErrInvalidSongID}
	handleSignup(rr, req, q)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCatalog_JSON(t *testing.T) {
	songs := []queue.Song{{ID: 1, Title: "Bohemian Rhapsody", Artist: "Queen"}}
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	rr := httptest.NewRecorder()
	handleCatalog(rr, req, &songsStub{stubQueue: &stubQueue{}, songs: songs})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"songs":[{"id":1,"title":"Bohemian Rhapsody","artist":"Queen"}]}`, rr.Body.String())
}

func TestHandleCatalog_EmptySongs(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	rr := httptest.NewRecorder()
	handleCatalog(rr, req, &songsStub{stubQueue: &stubQueue{}, songs: []queue.Song{}})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"songs":[]}`, rr.Body.String())
}

func TestQueueStatusData_ManyEntries(t *testing.T) {
	entries := []queue.Entry{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Carol"},
	}
	q := &stubQueue{entries: entries, signupsOpen: true}
	data := queueStatusData(context.Background(), q)
	assert.Equal(t, "Alice", data.Current.Name)
	// Next is always only the second entry, regardless of queue length
	assert.Equal(t, "Bob", data.Next.Name)
}