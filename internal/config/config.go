package config

import (
	"github.com/caarlos0/env/v11"

	"log"
)

type Config struct {
	Redis Redis
}

type Redis struct {
	Addr          string `env:"Redis_Address"`
	Password      string `env:"Redis_Password"`
	DB            int    `env:"Redis_DB"`
	StreamKey     string `env:"Redis_StreamKey"`
	Group         string `env:"Redis_Group"`
	ScheduledZSet string `env:"Redis_ScheduledZSet"`
	DLQStreamKey  string `env:"Redis_DLQStreamKey"`
}

func Load() *Config {
	var c Config
	if err := env.Parse(&c); err != nil {
		log.Fatal(err)
	}

	return &c
}
