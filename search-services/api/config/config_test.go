package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMustLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := []byte(`
log_level: INFO
api_server:
  address: ":9999"
  timeout: 3s
words_address: "words:1234"
update_address: "update:1235"
search_address: "search:1236"
token_ttl: 10s
search_concurrency: 5
search_rate: 50
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg := MustLoad(cfgPath)

	if cfg.LogLevel != "INFO" ||
		cfg.HTTPConfig.Address != ":9999" ||
		cfg.WordsAddress != "words:1234" ||
		cfg.UpdateAddress != "update:1235" ||
		cfg.SearchAddress != "search:1236" ||
		cfg.SearchConcurrency != 5 ||
		cfg.SearchRate != 50 {
		t.Fatalf("unexpected config %+v", cfg)
	}
	if cfg.HTTPConfig.Timeout != 3*time.Second || cfg.TokenTTL != 10*time.Second {
		t.Fatalf("unexpected durations: %+v", cfg)
	}
}

func TestMustLoad_FromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "ERROR")
	t.Setenv("API_ADDRESS", ":8888")
	t.Setenv("API_TIMEOUT", "7s")
	t.Setenv("WORDS_ADDRESS", "words:1111")
	t.Setenv("UPDATE_ADDRESS", "update:2222")
	t.Setenv("SEARCH_ADDRESS", "search:3333")
	t.Setenv("TOKEN_TTL", "20s")
	t.Setenv("SEARCH_CONCURRENCY", "2")
	t.Setenv("SEARCH_RATE", "12")

	cfg := MustLoad("") // nonexistent path forces env

	if cfg.LogLevel != "ERROR" || cfg.HTTPConfig.Address != ":8888" {
		t.Fatalf("unexpected values: %+v", cfg)
	}
	if cfg.HTTPConfig.Timeout != 7*time.Second || cfg.TokenTTL != 20*time.Second {
		t.Fatalf("unexpected durations: %+v", cfg)
	}
	if cfg.SearchConcurrency != 2 || cfg.SearchRate != 12 {
		t.Fatalf("unexpected numeric values: %+v", cfg)
	}
	if cfg.WordsAddress != "words:1111" || cfg.UpdateAddress != "update:2222" || cfg.SearchAddress != "search:3333" {
		t.Fatalf("unexpected addresses: %+v", cfg)
	}
}
