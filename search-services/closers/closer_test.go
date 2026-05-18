package closers

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

type stubCloser struct {
	err error
}

func (s stubCloser) Close() error {
	return s.err
}

func makeLogger(buf *bytes.Buffer) *slog.Logger {
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler)
}

func TestCloseOrLog_ErrorIsLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := makeLogger(&buf)

	CloseOrLog(stubCloser{err: errors.New("boom")}, logger)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "close failed") {
		t.Fatalf("expected log to contain 'close failed', got %q", logOutput)
	}
}

func TestCloseOrPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on close error")
		}
	}()

	CloseOrPanic(stubCloser{err: errors.New("panic please")})
}
