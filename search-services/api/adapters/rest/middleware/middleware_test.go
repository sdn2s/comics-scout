package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type stubVerifier struct {
	err error
}

func (s stubVerifier) Verify(string) error {
	return s.err
}

func TestAuth(t *testing.T) {
	called := false
	next := func(w http.ResponseWriter, r *http.Request) {
		called = true
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	Auth(next, stubVerifier{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing header, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token good")
	rr = httptest.NewRecorder()
	Auth(next, stubVerifier{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called {
		t.Fatalf("expected next handler to be called")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token bad")
	rr = httptest.NewRecorder()
	Auth(next, stubVerifier{err: assertErr{}}).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rr.Code)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }

func TestConcurrency(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	next := func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}

	handler := Concurrency(next, 1)

	// First request enters and blocks.
	go handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	<-started

	rr2 := httptest.NewRecorder()
	handler(rr2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when concurrency limit exceeded, got %d", rr2.Code)
	}

	close(release)
}

func TestRate(t *testing.T) {
	var calls atomic.Int32
	next := func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}

	handler := Rate(next, 1000) // small wait

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		done := make(chan struct{})
		go func() {
			handler(rr, req)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("handler blocked too long")
		}
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	}

	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}
