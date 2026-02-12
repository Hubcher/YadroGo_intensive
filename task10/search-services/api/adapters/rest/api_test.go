package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yadro.com/course/api/core"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

type mockAuthenticator struct {
	token        string
	err          error
	lastUser     string
	lastPassword string
}

func (m *mockAuthenticator) Login(user, password string) (string, error) {
	m.lastUser = user
	m.lastPassword = password
	return m.token, m.err
}

type mockUpdater struct {
	updateFn func(ctx context.Context) error
	statsFn  func(ctx context.Context) (core.UpdateStats, error)
	statusFn func(ctx context.Context) (core.UpdateStatus, error)
	dropFn   func(ctx context.Context) error
}

func (m *mockUpdater) Update(ctx context.Context) error {
	if m.updateFn == nil {
		return nil
	}
	return m.updateFn(ctx)
}

func (m *mockUpdater) Stats(ctx context.Context) (core.UpdateStats, error) {
	if m.statsFn == nil {
		return core.UpdateStats{}, nil
	}
	return m.statsFn(ctx)
}

func (m *mockUpdater) Status(ctx context.Context) (core.UpdateStatus, error) {
	if m.statusFn == nil {
		return core.StatusUpdateIdle, nil
	}
	return m.statusFn(ctx)
}

func (m *mockUpdater) Drop(ctx context.Context) error {
	if m.dropFn == nil {
		return nil
	}
	return m.dropFn(ctx)
}

type mockSearcher struct {
	searchFn      func(ctx context.Context, phrase string, limit int) ([]core.Comics, error)
	indexSearchFn func(ctx context.Context, phrase string, limit int) ([]core.Comics, error)
}

func (m *mockSearcher) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	if m.searchFn == nil {
		return nil, nil
	}
	return m.searchFn(ctx, phrase, limit)
}

func (m *mockSearcher) IndexSearch(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	if m.indexSearchFn == nil {
		return nil, nil
	}
	return m.indexSearchFn(ctx, phrase, limit)
}

func TestNewPingHandler_MixedReplies(t *testing.T) {
	log := newTestLogger()
	pingers := map[string]core.Pinger{
		"okService":  &mockPinger{err: nil},
		"badService": &mockPinger{err: errors.New("down-service")},
	}

	h := NewPingHandler(log, pingers)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp PingResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	assert.Equal(t, "ok", resp.Replies["okService"])
	assert.Equal(t, "unavailable", resp.Replies["badService"])
}

func TestNewLoginHandler_BadJSON(t *testing.T) {
	log := newTestLogger()
	auth := &mockAuthenticator{}

	h := NewLoginHandler(log, auth)

	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNewLoginHandler_Unauthorized(t *testing.T) {
	log := newTestLogger()
	auth := &mockAuthenticator{
		err: errors.New("invalid credentials"),
	}

	h := NewLoginHandler(log, auth)

	reqBody, _ := json.Marshal(LoginRequest{
		Name:     "admin",
		Password: "wrong",
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "admin", auth.lastUser)
	assert.Equal(t, "wrong", auth.lastPassword)
}

func TestNewLoginHandler_Success(t *testing.T) {
	log := newTestLogger()
	auth := &mockAuthenticator{
		token: "token123",
		err:   nil,
	}

	h := NewLoginHandler(log, auth)

	reqBody, _ := json.Marshal(LoginRequest{
		Name:     "admin",
		Password: "secret",
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "token123", rr.Body.String())
	assert.Equal(t, "admin", auth.lastUser)
	assert.Equal(t, "secret", auth.lastPassword)
}

func TestNewUpdateHandler_Success(t *testing.T) {
	log := newTestLogger()
	updater := &mockUpdater{
		updateFn: func(ctx context.Context) error {
			return nil
		},
	}

	h := NewUpdateHandler(log, updater)

	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestNewUpdateHandler_InternalError(t *testing.T) {
	log := newTestLogger()
	expErr := errors.New("some update error")
	updater := &mockUpdater{
		updateFn: func(ctx context.Context) error {
			return expErr
		},
	}

	h := NewUpdateHandler(log, updater)

	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), expErr.Error())
}

func TestNewUpdateStatsHandler_Error(t *testing.T) {
	log := newTestLogger()
	expErr := errors.New("stats failed")
	updater := &mockUpdater{
		statsFn: func(ctx context.Context) (core.UpdateStats, error) {
			return core.UpdateStats{}, expErr
		},
	}

	h := NewUpdateStatsHandler(log, updater)

	req := httptest.NewRequest(http.MethodGet, "/update/stats", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), expErr.Error())
}

func TestNewUpdateStatusHandler_Idle(t *testing.T) {
	log := newTestLogger()
	updater := &mockUpdater{
		statusFn: func(ctx context.Context) (core.UpdateStatus, error) {
			return core.StatusUpdateIdle, nil
		},
	}

	h := NewUpdateStatusHandler(log, updater)

	req := httptest.NewRequest(http.MethodGet, "/update/status", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp UpdateStatusResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "idle", resp.Status)
}

func TestNewUpdateStatusHandler_Running(t *testing.T) {
	log := newTestLogger()
	updater := &mockUpdater{
		statusFn: func(ctx context.Context) (core.UpdateStatus, error) {
			return core.StatusUpdateRunning, nil
		},
	}

	h := NewUpdateStatusHandler(log, updater)

	req := httptest.NewRequest(http.MethodGet, "/update/status", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp UpdateStatusResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "running", resp.Status)
}

func TestNewUpdateStatusHandler_Error(t *testing.T) {
	log := newTestLogger()
	expErr := errors.New("status error")
	updater := &mockUpdater{
		statusFn: func(ctx context.Context) (core.UpdateStatus, error) {
			return core.StatusUpdateIdle, expErr
		},
	}

	h := NewUpdateStatusHandler(log, updater)

	req := httptest.NewRequest(http.MethodGet, "/update/status", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), expErr.Error())
}

func TestNewDropHandler_Success(t *testing.T) {
	log := newTestLogger()
	updater := &mockUpdater{
		dropFn: func(ctx context.Context) error {
			return nil
		},
	}

	h := NewDropHandler(log, updater)

	req := httptest.NewRequest(http.MethodDelete, "/update/drop", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestNewDropHandler_Error(t *testing.T) {
	log := newTestLogger()
	expErr := errors.New("drop failed")
	updater := &mockUpdater{
		dropFn: func(ctx context.Context) error {
			return expErr
		},
	}

	h := NewDropHandler(log, updater)

	req := httptest.NewRequest(http.MethodDelete, "/update/drop", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), expErr.Error())
}

func TestNewSearchHandler_EmptyPhrase(t *testing.T) {
	log := newTestLogger()
	searcher := &mockSearcher{}

	h := NewSearchHandler(log, searcher)

	req := httptest.NewRequest(http.MethodGet, "/search?limit=5", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNewSearchHandler_InvalidLimit(t *testing.T) {
	log := newTestLogger()
	searcher := &mockSearcher{}

	h := NewSearchHandler(log, searcher)

	req := httptest.NewRequest(http.MethodGet, "/search?phrase=foo&limit=abc", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNewSearchHandler_BadArgumentsFromService(t *testing.T) {
	log := newTestLogger()
	searcher := &mockSearcher{
		searchFn: func(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
			return nil, core.ErrBadArguments
		},
	}

	h := NewSearchHandler(log, searcher)

	req := httptest.NewRequest(http.MethodGet, "/search?phrase=foo&limit=5", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), core.ErrBadArguments.Error())
}

func TestNewSearchHandler_InternalError(t *testing.T) {
	log := newTestLogger()
	expErr := errors.New("search failed")
	searcher := &mockSearcher{
		searchFn: func(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
			return nil, expErr
		},
	}

	h := NewSearchHandler(log, searcher)

	req := httptest.NewRequest(http.MethodGet, "/search?phrase=foo&limit=5", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal error")
}

func TestNewIndexSearchHandler_EmptyPhrase(t *testing.T) {
	log := newTestLogger()
	searcher := &mockSearcher{}

	h := NewIndexSearchHandler(log, searcher)

	req := httptest.NewRequest(http.MethodGet, "/indexsearch?limit=5", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNewIndexSearchHandler_InvalidLimit(t *testing.T) {
	log := newTestLogger()
	searcher := &mockSearcher{}

	h := NewIndexSearchHandler(log, searcher)

	req := httptest.NewRequest(http.MethodGet, "/indexsearch?phrase=foo&limit=-1", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
