package broker

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	log *slog.Logger
	nc  *nats.Conn
}

func NewPublisher(address string, log *slog.Logger) (*Publisher, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		log.Error("failed to connect to broker", "address", address, "error", err)
		return nil, err
	}

	return &Publisher{
		log: log,
		nc:  nc,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, subject string, payload []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := p.nc.Publish(subject, payload); err != nil {
		p.log.Error("failed to publish event", "subject", subject, "error", err)
		return err
	}

	return nil
}

func (p *Publisher) Close() error {
	p.nc.Close()
	return nil
}
