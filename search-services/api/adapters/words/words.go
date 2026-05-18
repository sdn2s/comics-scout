package words

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"yadro.com/course/api/core"
	wordspb "yadro.com/course/proto/words"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	wordsClient := wordspb.NewWordsClient(conn)

	client := &Client{
		log:    log,
		client: wordsClient,
		conn:   conn,
	}

	return client, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	request := &wordspb.WordsRequest{
		Phrase: phrase,
	}

	reply, err := c.client.Norm(ctx, request)
	if err != nil {
		code := status.Code(err)
		if code == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}

		return nil, err
	}

	words := reply.GetWords()

	return words, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
