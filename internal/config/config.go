package config

import (
	"encoding/json"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Mode            string `mapstructure:"MODE"`
	DatabaseFile    string `mapstructure:"DATABASE_FILE"`
	AdGuardUrl      string `mapstructure:"ADGUARD_URL"`
	AdGuardScheme   string `mapstructure:"ADGUARD_SCHEME"`
	AdGuardUsername string `mapstructure:"ADGUARD_USERNAME"`
	AdGuardPassword string `mapstructure:"ADGUARD_PASSWORD"`
	AdGuardLogging  bool   `mapstructure:"ADGUARD_LOGGING"`
	Annotation      string `mapstructure:"ANNOTATION"`
}

func (c Config) String() string {
	bytes, _ := json.MarshalIndent(c, "", "  ")
	return string(bytes)
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetDefault("MODE", "DEV")
	viper.SetDefault("DATABASE_FILE", "rules.db")
	viper.SetDefault("ADGUARD_URL", "localhost:8080")
	viper.SetDefault("ADGUARD_SCHEME", "http")
	viper.SetDefault("ADGUARD_USERNAME", "")
	viper.SetDefault("ADGUARD_PASSWORD", "")
	viper.SetDefault("ADGUARD_LOGGING", "false")
	viper.SetDefault("ANNOTATION", "external-dns.alpha.kubernetes.io/hostname")
	viper.AutomaticEnv()
	configErr := viper.ReadInConfig()
	if configErr != nil {
		log.Printf("Could not load config file: %v", configErr)
	}
	err = viper.Unmarshal(&config)
	return
}
