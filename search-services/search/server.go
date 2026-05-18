package main

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/service"
)

type server struct {
	searchpb.UnimplementedSearchServer
	service *service.Service
}

func (s *server) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *server) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	if req.GetPhrase() == "" {
		return nil, status.Error(codes.InvalidArgument, "phrase is required")
	}
	reply, err := s.service.Search(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return reply, nil
}

func (s *server) SearchIndex(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	return s.Search(ctx, req)
}
