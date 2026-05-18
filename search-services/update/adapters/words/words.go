package words

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/update/core"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		log.Error("failed to create words client connection", "address", address, "error", err)
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
	err := c.conn.Close()
	if err != nil {
		c.log.Error("failed to close words client connection", "error", err)
		return err
	}

	return nil
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

		c.log.Error("failed to normalize phrase", "phrase", phrase, "error", err)
		return nil, err
	}

	words := reply.GetWords()

	return words, nil
}
