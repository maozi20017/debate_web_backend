package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
}

type ServerConfig struct {
	Address string
}

type DBConfig struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     int
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./pkg/config")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
