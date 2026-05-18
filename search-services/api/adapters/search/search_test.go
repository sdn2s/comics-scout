package search

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
)

type stubSearchServer struct {
	searchpb.UnimplementedSearchServer
}

func (stubSearchServer) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (stubSearchServer) Search(_ context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	return &searchpb.SearchReply{
		Comics: []*searchpb.Comics{
			{Id: req.Limit, Url: req.Phrase + "/url"},
		},
	}, nil
}

func (stubSearchServer) SearchIndex(_ context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	return &searchpb.SearchReply{
		Comics: []*searchpb.Comics{
			{Id: req.Limit + 1, Url: req.Phrase + "/index"},
		},
	}, nil
}

func startSearchServer(t *testing.T) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	searchpb.RegisterSearchServer(server, stubSearchServer{})

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("serve stopped: %v", err)
		}
	}()

	return lis.Addr().String(), server.Stop
}

func TestClient_Search(t *testing.T) {
	addr, stop := startSearchServer(t)
	defer stop()

	client, err := NewClient(addr, nil)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	resp, err := client.Search(context.Background(), "linux", 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != 2 || resp[0].URL != "linux/url" {
		t.Fatalf("unexpected response %+v", resp)
	}

	respIndex, err := client.SearchIndex(context.Background(), "go", 1)
	if err != nil {
		t.Fatalf("search index failed: %v", err)
	}
	if len(respIndex) != 1 || respIndex[0].ID != 1 || respIndex[0].URL != "go/url" {
		t.Fatalf("unexpected index response %+v", respIndex)
	}
}

func TestClient_Ping(t *testing.T) {
	addr, stop := startSearchServer(t)
	defer stop()

	client, err := NewClient(addr, nil)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestFromProto(t *testing.T) {
	if res := fromProto(nil); res != nil {
		t.Fatalf("expected nil result")
	}

	resp := &searchpb.SearchReply{
		Comics: []*searchpb.Comics{{Id: 3, Url: "u"}},
	}
	res := fromProto(resp)
	if len(res) != 1 || res[0].ID != 3 || res[0].URL != "u" {
		t.Fatalf("unexpected conversion %+v", res)
	}
}
