package xkcd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"yadro.com/course/update/core"
)

func TestNewClient_EmptyURL(t *testing.T) {
	if _, err := NewClient("", time.Second, nil); err == nil {
		t.Fatalf("expected error for empty url")
	}
}

func TestGet_Special404(t *testing.T) {
	client, err := NewClient("http://example.com", time.Second, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	info, err := client.Get(context.Background(), 404)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if info.ID != 404 || info.Description != "404 Not Found" {
		t.Fatalf("unexpected info %+v", info)
	}
}

func TestGetAndLastID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info.0.json" {
			if _, err := fmt.Fprintf(w, `{"num":3,"img":"last.png","title":"last","safe_title":"safe","transcript":"t","alt":"a"}`); err != nil {
				t.Fatalf("write response: %v", err)
			}
			return
		}
		if strings.Contains(r.URL.Path, "/2/") {
			if _, err := fmt.Fprintf(w, `{"num":2,"img":"img.png","title":"Hello","safe_title":"Safe","transcript":"Transcript","alt":"Alt"}`); err != nil {
				t.Fatalf("write response: %v", err)
			}
			return
		}
		http.NotFound(w, r)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client, err := NewClient(server.URL, time.Second, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	info, err := client.Get(context.Background(), 2)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if info.ID != 2 || info.URL != "img.png" {
		t.Fatalf("unexpected info %+v", info)
	}
	if info.Description == "" || info.Description != "Hello Safe Transcript Alt" {
		t.Fatalf("unexpected description %q", info.Description)
	}

	lastID, err := client.LastID(context.Background())
	if err != nil {
		t.Fatalf("last id failed: %v", err)
	}
	if lastID != 3 {
		t.Fatalf("unexpected last id %d", lastID)
	}
}

func TestGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client, err := NewClient(server.URL, time.Second, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Get(context.Background(), 1); err != core.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
