package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Address string        `yaml:"address" env:"HTTP_SERVER_ADDRESS" env-default:":8080"`
	Timeout time.Duration `yaml:"timeout" env:"HTTP_SERVER_TIMEOUT" env-default:"5s"`
}

type Config struct {
	LogLevel     string     `yaml:"log_level" env:"LOG_LEVEL" env-default:"INFO"`
	WordsAddress string     `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"localhost:8081"`
	HTTPServer   HTTPServer `yaml:"http_server"`
}

func MustLoad(configPath string) Config {
	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}

	return cfg
}
