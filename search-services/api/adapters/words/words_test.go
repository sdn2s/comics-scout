package words

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	wordspb "yadro.com/course/proto/words"
)

type stubWordsServer struct {
	wordspb.UnimplementedWordsServer
	err error
}

func (s stubWordsServer) Norm(_ context.Context, req *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &wordspb.WordsReply{Words: []string{req.Phrase, "norm"}}, nil
}

func (s stubWordsServer) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
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

func TestClient_NormAndPing(t *testing.T) {
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

	words, err := client.Norm(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("norm failed: %v", err)
	}
	if len(words) != 2 || words[0] != "Hello" {
		t.Fatalf("unexpected words %v", words)
	}

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
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
