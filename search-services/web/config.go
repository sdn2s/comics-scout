package main

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Address        string        `yaml:"address" env:"WEB_ADDRESS" env-default:":8080"`
	APIBaseURL     string        `yaml:"api_base_url" env:"API_BASE_URL" env-default:"http://api:8080"`
	TokenCookie    string        `yaml:"token_cookie" env:"TOKEN_COOKIE" env-default:"comic_token"`
	TokenTTL       time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"2m"`
	RequestTimeout time.Duration `yaml:"request_timeout" env:"REQUEST_TIMEOUT" env-default:"8s"`
	LogLevel       string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"INFO"`
}

func mustLoadConfig(path string) Config {
	var cfg Config

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				log.Fatalf("cannot read config %q: %v", path, err)
			}
			return cfg
		}
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read config from env: %v", err)
	}

	return cfg
}
