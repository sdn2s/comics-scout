package grpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

type stubUpdater struct {
	updateErr error
	stats     core.ServiceStats
	statsErr  error
	status    core.ServiceStatus
	dropErr   error
}

func (s stubUpdater) Update(context.Context) error {
	return s.updateErr
}

func (s stubUpdater) Stats(context.Context) (core.ServiceStats, error) {
	return s.stats, s.statsErr
}

func (s stubUpdater) Status(context.Context) core.ServiceStatus {
	return s.status
}

func (s stubUpdater) Drop(context.Context) error {
	return s.dropErr
}

func TestServer_Status(t *testing.T) {
	server := NewServer(stubUpdater{status: core.StatusIdle})
	resp, err := server.Status(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if resp.Status != updatepb.Status_STATUS_IDLE {
		t.Fatalf("unexpected status %v", resp.Status)
	}

	server = NewServer(stubUpdater{status: "weird"})
	if _, err := server.Status(context.Background(), &emptypb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error for unknown status, got %v", err)
	}
}

func TestServer_Update(t *testing.T) {
	server := NewServer(stubUpdater{updateErr: core.ErrAlreadyExists})
	if _, err := server.Update(context.Background(), &emptypb.Empty{}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected already exists error, got %v", err)
	}

	server = NewServer(stubUpdater{updateErr: errors.New("boom")})
	if _, err := server.Update(context.Background(), &emptypb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestServer_StatsAndDrop(t *testing.T) {
	server := NewServer(stubUpdater{
		stats: core.ServiceStats{
			DBStats: core.DBStats{
				WordsTotal:    2,
				WordsUnique:   1,
				ComicsFetched: 3,
			},
			ComicsTotal: 4,
		},
	})

	stats, err := server.Stats(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.WordsTotal != 2 || stats.ComicsTotal != 4 {
		t.Fatalf("unexpected stats %+v", stats)
	}

	server = NewServer(stubUpdater{dropErr: errors.New("fail")})
	if _, err := server.Drop(context.Background(), &emptypb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}
