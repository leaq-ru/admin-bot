package config

import "github.com/kelseyhightower/envconfig"

func NewConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return Config{}, err
	}

	cfg.ServiceName = "admin-bot"
	return cfg, nil
}
