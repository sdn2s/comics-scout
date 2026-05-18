package update

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	conn   *grpc.ClientConn
	client updatepb.UpdateClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to create update client", "address", address, "error", err)
		return nil, err
	}

	return &Client{
		log:    log,
		conn:   conn,
		client: updatepb.NewUpdateClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("update ping failed", "error", err)
	}
	return err
}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return core.ErrAlreadyExists
		}
		c.log.Error("update RPC failed", "error", err)
		return err
	}
	return nil
}

func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	reply, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("stats RPC failed", "error", err)
		return core.UpdateStats{}, err
	}

	stats := core.UpdateStats{
		WordsTotal:    int(reply.WordsTotal),
		WordsUnique:   int(reply.WordsUnique),
		ComicsFetched: int(reply.ComicsFetched),
		ComicsTotal:   int(reply.ComicsTotal),
	}

	if stats.ComicsFetched > 0 {
		stats.ComicsTotal = stats.ComicsFetched
	}

	return stats, nil
}

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	reply, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("status RPC failed", "error", err)
		return "", err
	}

	switch reply.Status {
	case updatepb.Status_STATUS_IDLE:
		return core.UpdateStatus("idle"), nil
	case updatepb.Status_STATUS_RUNNING:
		return core.UpdateStatus("running"), nil
	default:
		c.log.Error("unknown status from update service", "status", reply.Status)
		return "", fmt.Errorf("unknown status: %v", reply.Status)
	}
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("drop RPC failed", "error", err)
		return err
	}
	return nil
}
