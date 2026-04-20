package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	PORT           string
	ALLOWED_ORIGIN string
}

func LoadConfig() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, using default env")
	}

	return &Config{
		PORT:           viper.GetString("PORT"),
		ALLOWED_ORIGIN: viper.GetString("ALLOWED_ORIGIN"),
	}
}
