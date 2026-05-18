package words

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/update/core"
)

type stubWordsServer struct {
	wordspb.UnimplementedWordsServer
	err error
}

func (s stubWordsServer) Norm(_ context.Context, req *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &wordspb.WordsReply{Words: []string{req.Phrase, "ok"}}, nil
}

func startWordsServer(t *testing.T, srv wordspb.WordsServer) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	wordspb.RegisterWordsServer(server, srv)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("serve stopped: %v", err)
		}
	}()
	return lis.Addr().String(), server.Stop
}

func TestClient_Norm(t *testing.T) {
	addr, stop := startWordsServer(t, stubWordsServer{})
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

	words, err := client.Norm(context.Background(), "phrase")
	if err != nil {
		t.Fatalf("norm failed: %v", err)
	}
	if len(words) != 2 || words[0] != "phrase" {
		t.Fatalf("unexpected words %v", words)
	}
}

func TestClient_NormBadArguments(t *testing.T) {
	addr, stop := startWordsServer(t, stubWordsServer{
		err: status.Error(codes.ResourceExhausted, "too long"),
	})
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

	if _, err := client.Norm(context.Background(), "bad"); err != core.ErrBadArguments {
		t.Fatalf("expected ErrBadArguments, got %v", err)
	}
}
