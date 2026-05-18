package core

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

type fakeDB struct {
	ids      []int
	added    []Comics
	addErr   error
	stats    DBStats
	statsErr error
	dropErr  error
}

func (f *fakeDB) Add(_ context.Context, comics Comics) error {
	f.added = append(f.added, comics)
	return f.addErr
}

func (f *fakeDB) Stats(context.Context) (DBStats, error) {
	return f.stats, f.statsErr
}

func (f *fakeDB) Drop(context.Context) error {
	return f.dropErr
}

func (f *fakeDB) IDs(context.Context) ([]int, error) {
	return f.ids, nil
}

type fakeXKCD struct {
	lastID  int
	lastErr error
	infos   map[int]XKCDInfo
	errs    map[int]error
}

func (f fakeXKCD) Get(_ context.Context, id int) (XKCDInfo, error) {
	if err, ok := f.errs[id]; ok {
		return XKCDInfo{}, err
	}
	if info, ok := f.infos[id]; ok {
		return info, nil
	}
	return XKCDInfo{}, ErrNotFound
}

func (f fakeXKCD) LastID(context.Context) (int, error) {
	return f.lastID, f.lastErr
}

type fakeWords struct {
	results map[string][]string
	err     error
}

func (f fakeWords) Norm(_ context.Context, phrase string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if res, ok := f.results[phrase]; ok {
		return res, nil
	}
	return []string{phrase}, nil
}

type fakePublisher struct {
	subject string
	payload []byte
	err     error
}

func (f *fakePublisher) Publish(_ context.Context, subject string, payload []byte) error {
	f.subject = subject
	f.payload = payload
	return f.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func TestNewService_InvalidConcurrency(t *testing.T) {
	if _, err := NewService(discardLogger(), nil, nil, nil, nil, 0); err == nil {
		t.Fatalf("expected error for invalid concurrency")
	}
}

func TestServiceUpdate_Success(t *testing.T) {
	db := &fakeDB{ids: []int{2}}
	xkcd := fakeXKCD{
		lastID: 3,
		infos: map[int]XKCDInfo{
			1: {ID: 1, URL: "u1", Description: "d1"},
			2: {ID: 2, URL: "u2", Description: "d2"},
			3: {ID: 3, URL: "u3", Description: "d3"},
		},
	}
	words := fakeWords{
		results: map[string][]string{
			"d1": {"w1"},
			"d3": {"w3"},
		},
	}
	pub := &fakePublisher{}

	service, err := NewService(discardLogger(), db, xkcd, words, pub, 2)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if err := service.Update(context.Background()); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(db.added) != 2 {
		t.Fatalf("expected 2 records inserted, got %d", len(db.added))
	}
	if pub.subject != "xkcd.db.updated" {
		t.Fatalf("expected publish call, got %q", pub.subject)
	}
	if service.Status(context.Background()) != StatusIdle {
		t.Fatalf("expected idle status after update")
	}
}

func TestServiceUpdate_AlreadyRunning(t *testing.T) {
	service, _ := NewService(discardLogger(), &fakeDB{}, fakeXKCD{lastID: 1, infos: map[int]XKCDInfo{}}, fakeWords{}, nil, 1)
	service.status = StatusRunning

	if err := service.Update(context.Background()); err != ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestService_StatusAndStats(t *testing.T) {
	db := &fakeDB{stats: DBStats{WordsTotal: 2}}
	xkcd := fakeXKCD{lastID: 5}

	service, _ := NewService(discardLogger(), db, xkcd, fakeWords{}, nil, 1)

	if !service.setStatusRunning() {
		t.Fatalf("expected status to switch to running")
	}
	if service.setStatusRunning() {
		t.Fatalf("should not allow running twice")
	}
	service.setStatusIdle()

	stats, err := service.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.ComicsTotal != 5 || stats.WordsTotal != 2 {
		t.Fatalf("unexpected stats %+v", stats)
	}
}

func TestServiceDrop(t *testing.T) {
	db := &fakeDB{}
	pub := &fakePublisher{}
	service, _ := NewService(discardLogger(), db, fakeXKCD{lastID: 1}, fakeWords{}, pub, 1)

	if err := service.Drop(context.Background()); err != nil {
		t.Fatalf("drop failed: %v", err)
	}
	if pub.subject != "xkcd.db.dropped" {
		t.Fatalf("expected drop event, got %q", pub.subject)
	}
}
