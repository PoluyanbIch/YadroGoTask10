package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel  string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address   string `yaml:"aaa_address" env:"AAA_ADDRESS" env-default:"localhost:80"`
	DBAddress string `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`

	AccessSecret   string        `yaml:"access_secret" env:"ACCESS_SECRET" env-default:"ipawjfpiajwf"`
	RefreshSecret  string        `yaml:"refresh_secret" env:"REFRESH_SECRET" env-default:"opajgkjsegseg"`
	AccessTokenTTL time.Duration `yaml:"access_ttl" env:"ACCESS_TTL" env-default:"5m"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
