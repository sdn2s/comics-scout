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
	"strings"
	"testing"

	aaa "yadro.com/course/api/adapters/aaa"
	"yadro.com/course/api/core"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

type stubUpdater struct {
	updateErr error
	stats     core.UpdateStats
	statsErr  error
	status    core.UpdateStatus
	statusErr error
	dropErr   error
}

func (s stubUpdater) Update(context.Context) error {
	return s.updateErr
}

func (s stubUpdater) Stats(context.Context) (core.UpdateStats, error) {
	return s.stats, s.statsErr
}

func (s stubUpdater) Status(context.Context) (core.UpdateStatus, error) {
	return s.status, s.statusErr
}

func (s stubUpdater) Drop(context.Context) error {
	return s.dropErr
}

type stubSearcher struct {
	results []core.Comics
	err     error
}

func (s stubSearcher) Search(context.Context, string, int) ([]core.Comics, error) {
	return s.results, s.err
}

func (s stubSearcher) SearchIndex(context.Context, string, int) ([]core.Comics, error) {
	return s.results, s.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEncodeReply(t *testing.T) {
	var buf bytes.Buffer
	if err := encodeReply(&buf, map[string]int{"a": 1}); err != nil {
		t.Fatalf("encodeReply failed: %v", err)
	}
	if !strings.Contains(buf.String(), "\n") {
		t.Fatalf("expected pretty-printed JSON, got %q", buf.String())
	}
}

func TestLoginHandler(t *testing.T) {
	auth, _ := aaa.New(0, discardLogger())
	handler := NewLoginHandler(discardLogger(), auth)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"name":"admin","password":"password"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected token in response")
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("unexpected content type %q", ct)
	}
}

func TestLoginHandlerErrors(t *testing.T) {
	auth, _ := aaa.New(0, discardLogger())
	handler := NewLoginHandler(discardLogger(), auth)

	tests := []struct {
		name     string
		body     string
		expected int
	}{
		{"bad json", "{", http.StatusBadRequest},
		{"wrong creds", `{"name":"admin","password":"wrong"}`, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, rr.Code)
			}
		})
	}
}

func TestPingHandler(t *testing.T) {
	handler := NewPingHandler(discardLogger(), map[string]core.Pinger{
		"ok":    stubPinger{},
		"fault": stubPinger{err: errors.New("boom")},
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rr.Code)
	}
	var resp PingResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode ping reply: %v", err)
	}
	if resp.Replies["ok"] != "ok" || resp.Replies["fault"] != "unavailable" {
		t.Fatalf("unexpected replies %+v", resp.Replies)
	}
}

func TestUpdateHandler(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		handler := NewUpdateHandler(discardLogger(), stubUpdater{})
		req := httptest.NewRequest(http.MethodPost, "/update", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("already exists", func(t *testing.T) {
		handler := NewUpdateHandler(discardLogger(), stubUpdater{updateErr: core.ErrAlreadyExists})
		req := httptest.NewRequest(http.MethodPost, "/update", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", rr.Code)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		handler := NewUpdateHandler(discardLogger(), stubUpdater{updateErr: errors.New("fail")})
		req := httptest.NewRequest(http.MethodPost, "/update", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestUpdateStatsHandler(t *testing.T) {
	updater := stubUpdater{
		stats: core.UpdateStats{
			WordsTotal:    10,
			WordsUnique:   5,
			ComicsFetched: 3,
			ComicsTotal:   7,
		},
	}

	handler := NewUpdateStatsHandler(discardLogger(), updater)
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rr.Code)
	}
	body := rr.Body.String()
	for _, substr := range []string{`"words_total": 10`, `"comics_total": 7`} {
		if !strings.Contains(body, substr) {
			t.Fatalf("expected %q in %q", substr, body)
		}
	}
}

func TestUpdateStatusHandler(t *testing.T) {
	handler := NewUpdateStatusHandler(discardLogger(), stubUpdater{status: core.StatusUpdateRunning})
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"running"`) {
		t.Fatalf("unexpected body %q", rr.Body.String())
	}
}

func TestDropHandler(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		handler := NewDropHandler(discardLogger(), stubUpdater{})
		req := httptest.NewRequest(http.MethodDelete, "/db", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		handler := NewDropHandler(discardLogger(), stubUpdater{dropErr: errors.New("fail")})
		req := httptest.NewRequest(http.MethodDelete, "/db", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestSearchHandlers(t *testing.T) {
	log := discardLogger()

	t.Run("bad input", func(t *testing.T) {
		handler := NewSearchHandler(log, stubSearcher{})
		req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewSearchHandler(log, stubSearcher{err: core.ErrNotFound})
		req := httptest.NewRequest(http.MethodGet, "/api/search?phrase=linux", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), `"total":0`) {
			t.Fatalf("unexpected body %q", rr.Body.String())
		}
	})

	t.Run("bad limit", func(t *testing.T) {
		handler := NewSearchHandler(log, stubSearcher{})
		req := httptest.NewRequest(http.MethodGet, "/api/search?phrase=linux&limit=abc", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		results := []core.Comics{
			{ID: 1, URL: "https://example.com/1"},
			{ID: 2, URL: "https://example.com/2"},
		}
		handler := NewSearchHandler(log, stubSearcher{results: results})
		req := httptest.NewRequest(http.MethodGet, "/api/search?phrase=hello&limit=2", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		body := rr.Body.String()
		if !strings.Contains(body, `"total":2`) || !strings.Contains(body, "example.com/1") {
			t.Fatalf("unexpected body %q", body)
		}
	})
}
