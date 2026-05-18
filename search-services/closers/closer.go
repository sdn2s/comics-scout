package closers

import (
	"io"
	"log/slog"
)

func CloseOrLog(closer io.Closer, logger *slog.Logger) {
	err := closer.Close()
	if err != nil {
		logger.Error("close failed", "error", err)
	}
}

func CloseOrPanic(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		panic(err)
	}
}
