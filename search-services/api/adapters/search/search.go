package search

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log  *slog.Logger
	conn *grpc.ClientConn
	api  searchpb.SearchClient
}

// nolint:staticcheck
func NewClient(addr string, log *slog.Logger) (*Client, error) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to search service %s: %w", addr, err)
	}

	return &Client{
		log:  log,
		conn: conn,
		api:  searchpb.NewSearchClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(_ context.Context) error {
	return nil
}

func (c *Client) doSearch(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	req := &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	}

	resp, err := c.api.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	return fromProto(resp), nil
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	return c.doSearch(ctx, phrase, limit)
}

func (c *Client) SearchIndex(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	return c.doSearch(ctx, phrase, limit)
}

func fromProto(resp *searchpb.SearchReply) []core.Comics {
	if resp == nil || len(resp.Comics) == 0 {
		return nil
	}

	res := make([]core.Comics, 0, len(resp.Comics))
	for _, c := range resp.Comics {
		res = append(res, core.Comics{
			ID:  int(c.Id),
			URL: c.Url,
		})
	}
	return res
}
