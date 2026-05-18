package update

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type stubUpdateServer struct {
	updatepb.UnimplementedUpdateServer
	updateErr error
	stats     *updatepb.StatsReply
	statsErr  error
	status    updatepb.Status
	dropErr   error
}

func (s stubUpdateServer) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s stubUpdateServer) Update(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.updateErr
}

func (s stubUpdateServer) Stats(context.Context, *emptypb.Empty) (*updatepb.StatsReply, error) {
	return s.stats, s.statsErr
}

func (s stubUpdateServer) Status(context.Context, *emptypb.Empty) (*updatepb.StatusReply, error) {
	return &updatepb.StatusReply{Status: s.status}, nil
}

func (s stubUpdateServer) Drop(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.dropErr
}

func startUpdateServer(t *testing.T, srv updatepb.UpdateServer) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	updatepb.RegisterUpdateServer(server, srv)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("serve stopped: %v", err)
		}
	}()
	return lis.Addr().String(), server.Stop
}

func TestClient_UpdateAndPing(t *testing.T) {
	srv := stubUpdateServer{
		updateErr: status.Error(codes.AlreadyExists, "running"),
	}
	addr, stop := startUpdateServer(t, srv)
	defer stop()

	client, err := NewClient(addr, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	if err := client.Update(context.Background()); err != core.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestClient_Stats(t *testing.T) {
	srv := stubUpdateServer{
		stats: &updatepb.StatsReply{
			WordsTotal:    10,
			WordsUnique:   5,
			ComicsFetched: 3,
			ComicsTotal:   9,
		},
	}
	addr, stop := startUpdateServer(t, srv)
	defer stop()

	client, err := NewClient(addr, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	stats, err := client.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.ComicsTotal != stats.ComicsFetched {
		t.Fatalf("expected ComicsTotal to equal fetched when fetched>0, got %+v", stats)
	}
}

func TestClient_Status(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	addr, stop := startUpdateServer(t, stubUpdateServer{status: updatepb.Status_STATUS_RUNNING})
	defer stop()

	client, err := NewClient(addr, log)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	statusValue, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if statusValue != core.UpdateStatus("running") {
		t.Fatalf("unexpected status %q", statusValue)
	}

	// Unknown status should error.
	addr2, stop2 := startUpdateServer(t, stubUpdateServer{status: updatepb.Status_STATUS_UNSPECIFIED})
	defer stop2()

	client2, err := NewClient(addr2, log)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client2.Close(); err != nil {
			t.Fatalf("close client2: %v", err)
		}
	}()

	if _, err := client2.Status(context.Background()); err == nil {
		t.Fatalf("expected error on unknown status")
	}
}

func TestClient_Drop(t *testing.T) {
	srv := stubUpdateServer{}
	addr, stop := startUpdateServer(t, srv)
	defer stop()

	client, err := NewClient(addr, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	if err := client.Drop(context.Background()); err != nil {
		t.Fatalf("drop failed: %v", err)
	}
}
