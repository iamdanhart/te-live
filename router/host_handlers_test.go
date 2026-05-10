package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/queue"
	"github.com/stretchr/testify/assert"
)

// TestMain changes to the repo root so the dev template loader can find
// grab_templates/templates/* relative to the working directory.
func TestMain(m *testing.M) {
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// noopLimiter passes all requests through without rate limiting.
type noopLimiter struct{}

func (noopLimiter) Limit(next http.Handler) http.Handler { return next }

// noopCSRF passes all requests through without CSRF checks.
func noopCSRF(next http.Handler) http.Handler { return next }

// hostStub is a configurable queue.Queue stub for host route tests.
type hostStub struct {
	entries     []queue.Entry
	performed   []queue.PerformedSong
	signupsOpen bool

	completeCurrentSongErr error
	moveCurrentToBottomErr error
	removeCurrentErr       error
	addSongToFirstErr      error
	moveEntryErr           error

	completeCurrentSongCalled bool
	moveCurrentToBottomCalled bool
	removeCurrentCalled       bool
	addSongToFirstCalled      bool
	moveEntryCalled           bool
	toggleSignupsCalled       bool
}

func (s *hostStub) Entries(context.Context) []queue.Entry              { return s.entries }
func (s *hostStub) SignupsOpen(context.Context) bool                   { return s.signupsOpen }
func (s *hostStub) Songs(context.Context) []queue.Song                 { return nil }
func (s *hostStub) HasName(context.Context, string) bool               { return false }
func (s *hostStub) AuthenticateHost(context.Context, string) bool      { return true }
func (s *hostStub) Performed(context.Context) []queue.PerformedSong    { return s.performed }
func (s *hostStub) Add(context.Context, string, []int) error           { return nil }

func (s *hostStub) ToggleSignups(context.Context) bool {
	s.toggleSignupsCalled = true
	return s.signupsOpen
}
func (s *hostStub) CompleteCurrentSong(_ context.Context, _ string, _ int) error {
	s.completeCurrentSongCalled = true
	return s.completeCurrentSongErr
}
func (s *hostStub) MoveCurrentToBottom(context.Context) error {
	s.moveCurrentToBottomCalled = true
	return s.moveCurrentToBottomErr
}
func (s *hostStub) RemoveCurrent(context.Context) error {
	s.removeCurrentCalled = true
	return s.removeCurrentErr
}
func (s *hostStub) AddSongToFirst(_ context.Context, _ int) error {
	s.addSongToFirstCalled = true
	return s.addSongToFirstErr
}
func (s *hostStub) MoveEntry(_ context.Context, _ int, _ int) error {
	s.moveEntryCalled = true
	return s.moveEntryErr
}

func newHostMux(q *hostStub) *http.ServeMux {
	mux := http.NewServeMux()
	registerHostRoutes(mux, config.Props{EnforceAdminAuth: false}, q, noopLimiter{}, noopCSRF)
	return mux
}

func postForm(mux *http.ServeMux, path string, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

// --- POST /host/performed ---

func TestHostPerformed_InvalidSongID(t *testing.T) {
	rr := postForm(newHostMux(&hostStub{}), "/host/performed", url.Values{
		"singer":  {"Alice"},
		"song_id": {"notanumber"},
	})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHostPerformed_CompleteCurrentSongError(t *testing.T) {
	q := &hostStub{completeCurrentSongErr: errors.New("db error")}
	rr := postForm(newHostMux(q), "/host/performed", url.Values{
		"singer":  {"Alice"},
		"song_id": {"1"},
	})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.True(t, q.completeCurrentSongCalled)
}

func TestHostPerformed_MoveCurrentToBottomError(t *testing.T) {
	q := &hostStub{moveCurrentToBottomErr: errors.New("db error")}
	rr := postForm(newHostMux(q), "/host/performed", url.Values{
		"singer":  {"Alice"},
		"song_id": {"1"},
	})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.True(t, q.completeCurrentSongCalled)
	assert.True(t, q.moveCurrentToBottomCalled)
}

// --- POST /host/add-song ---

func TestHostAddSong_InvalidSongID(t *testing.T) {
	rr := postForm(newHostMux(&hostStub{}), "/host/add-song", url.Values{
		"song_id": {"notanumber"},
	})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHostAddSong_Error(t *testing.T) {
	q := &hostStub{addSongToFirstErr: errors.New("db error")}
	rr := postForm(newHostMux(q), "/host/add-song", url.Values{"song_id": {"1"}})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.True(t, q.addSongToFirstCalled)
}

func TestHostAddSong_Success(t *testing.T) {
	q := &hostStub{}
	rr := postForm(newHostMux(q), "/host/add-song", url.Values{"song_id": {"1"}})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, q.addSongToFirstCalled)
}

// --- POST /host/remove ---

func TestHostRemove_Error(t *testing.T) {
	q := &hostStub{removeCurrentErr: errors.New("db error")}
	rr := postForm(newHostMux(q), "/host/remove", url.Values{})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.True(t, q.removeCurrentCalled)
}

func TestHostRemove_Success(t *testing.T) {
	q := &hostStub{}
	rr := postForm(newHostMux(q), "/host/remove", url.Values{})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, q.removeCurrentCalled)
}

// --- POST /host/skip ---

func TestHostSkip_Error(t *testing.T) {
	q := &hostStub{moveCurrentToBottomErr: errors.New("db error")}
	rr := postForm(newHostMux(q), "/host/skip", url.Values{})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.True(t, q.moveCurrentToBottomCalled)
}

func TestHostSkip_Success(t *testing.T) {
	q := &hostStub{}
	rr := postForm(newHostMux(q), "/host/skip", url.Values{})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, q.moveCurrentToBottomCalled)
}

// --- POST /host/move ---

func TestHostMove_InvalidID(t *testing.T) {
	rr := postForm(newHostMux(&hostStub{}), "/host/move", url.Values{
		"id":       {"notanumber"},
		"after_id": {"0"},
	})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHostMove_InvalidAfterID(t *testing.T) {
	rr := postForm(newHostMux(&hostStub{}), "/host/move", url.Values{
		"id":       {"1"},
		"after_id": {"notanumber"},
	})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHostMove_Error(t *testing.T) {
	q := &hostStub{moveEntryErr: errors.New("db error")}
	rr := postForm(newHostMux(q), "/host/move", url.Values{
		"id":       {"1"},
		"after_id": {"0"},
	})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.True(t, q.moveEntryCalled)
}

func TestHostMove_Success(t *testing.T) {
	q := &hostStub{}
	rr := postForm(newHostMux(q), "/host/move", url.Values{
		"id":       {"1"},
		"after_id": {"0"},
	})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, q.moveEntryCalled)
}

// --- POST /signups/toggle ---

func TestHostToggleSignups_JSONContentType(t *testing.T) {
	q := &hostStub{signupsOpen: true}
	rr := postForm(newHostMux(q), "/signups/toggle", url.Values{})
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"signups_open":true}`, rr.Body.String())
	assert.True(t, q.toggleSignupsCalled)
}

func TestHostToggleSignups_ReturnsFalse(t *testing.T) {
	q := &hostStub{signupsOpen: false}
	rr := postForm(newHostMux(q), "/signups/toggle", url.Values{})
	assert.JSONEq(t, `{"signups_open":false}`, rr.Body.String())
}