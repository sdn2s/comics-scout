package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	statusValue := s.service.Status(ctx)

	var statusProto updatepb.Status
	switch statusValue {
	case core.StatusIdle:
		statusProto = updatepb.Status_STATUS_IDLE
	case core.StatusRunning:
		statusProto = updatepb.Status_STATUS_RUNNING
	default:
		return nil, status.Error(codes.Internal, "unknown status")
	}

	return &updatepb.StatusReply{Status: statusProto}, nil
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		if err == core.ErrAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	stats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &updatepb.StatsReply{
		WordsTotal:    int64(stats.WordsTotal),
		WordsUnique:   int64(stats.WordsUnique),
		ComicsTotal:   int64(stats.ComicsTotal),
		ComicsFetched: int64(stats.ComicsFetched),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}
