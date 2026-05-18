package broker

import (
	"context"
	"log/slog"
	"testing"

	server "github.com/nats-io/nats-server/v2/server"
	testserver "github.com/nats-io/nats-server/v2/test"
)

func TestPublish_ContextCanceled(t *testing.T) {
	p := &Publisher{log: slog.Default()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := p.Publish(ctx, "subject", []byte("data")); err == nil {
		t.Fatalf("expected error for canceled context")
	}
}

func TestNewPublisherAndPublish(t *testing.T) {
	ns := testserver.RunServer(&server.Options{Port: -1})
	defer ns.Shutdown()

	publisher, err := NewPublisher(ns.ClientURL(), slog.Default())
	if err != nil {
		t.Fatalf("new publisher failed: %v", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Fatalf("close publisher: %v", err)
		}
	}()

	if err := publisher.Publish(context.Background(), "subject", []byte("payload")); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
}
