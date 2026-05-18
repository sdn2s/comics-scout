package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMustLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(path, []byte(`
log_level: INFO
update_address: ":7777"
db_address: "postgres://user:pass@localhost/db"
broker_address: "nats://localhost:4222"
words_address: "words:123"
api_address: ":9000"
xkcd:
  url: "https://xkcd.com"
  timeout: 2s
  concurrency: 3
`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg := MustLoad(path)

	if cfg.Address != ":7777" || cfg.DBAddress != "postgres://user:pass@localhost/db" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.XKCD.Timeout != 2*time.Second || cfg.XKCD.Concurrency != 3 {
		t.Fatalf("unexpected timeout/concurrency: %+v", cfg)
	}
	if cfg.LogLevel != "INFO" || cfg.BrokerAddress != "nats://localhost:4222" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestMustLoadFromEnv(t *testing.T) {
	t.Setenv("UPDATE_ADDRESS", ":1234")
	t.Setenv("DB_ADDRESS", "db")
	t.Setenv("BROKER_ADDRESS", "nats://example:4222")
	t.Setenv("WORDS_ADDRESS", "words:10")
	t.Setenv("API_ADDRESS", ":80")
	t.Setenv("XKCD_URL", "https://example.com")
	t.Setenv("XKCD_TIMEOUT", "5s")
	t.Setenv("XKCD_CONCURRENCY", "2")

	cfg := MustLoad("")

	if cfg.Address != ":1234" || cfg.DBAddress != "db" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.XKCD.Timeout != 5*time.Second || cfg.XKCD.Concurrency != 2 {
		t.Fatalf("unexpected xkcd section: %+v", cfg.XKCD)
	}
	if cfg.BrokerAddress != "nats://example:4222" || cfg.WordsAddress != "words:10" {
		t.Fatalf("unexpected addresses: %+v", cfg)
	}
}
