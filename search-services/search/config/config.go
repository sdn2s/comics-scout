package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Address     string `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:":8080"`
	DBAddress   string `yaml:"db_address" env:"DB_ADDRESS" env-default:"postgres://postgres:password@postgres:5432/postgres?sslmode=disable"`
	WordsTarget string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"words:8080"`
}

func MustLoad(path string) Config {
	var cfg Config

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				log.Fatalf("cannot read config %q: %s", path, err)
			}
			return cfg
		}
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read config from env: %s", err)
	}

	return cfg
}
