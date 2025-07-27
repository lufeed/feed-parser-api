package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
)

var (
	conf *AppConfig
)

func Initialize() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		log.Fatal("CONFIG_FILE environment variable is not set")
	}

	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(fmt.Errorf("error reading config file: %w", err))
	}

	if err := viper.Unmarshal(&conf); err != nil {
		log.Fatal(fmt.Errorf("error unmarshalling config: %w", err))
	}

}

func GetConfig() *AppConfig {
	return conf
}
