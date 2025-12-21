package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	Auth struct {
		Enable bool   `mapstructure:"enable"`
		Token  string `mapstructure:"token"`
	} `mapstructure:"auth"`
	MCSM struct {
		URL    string `mapstructure:"url"`
		APIKey string `mapstructure:"apikey"`
	} `mapstructure:"mcsm"`
	App struct {
		ExternalURL string `mapstructure:"external_url"`
	} `mapstructure:"app"`
	DBPath string
}

func Load() *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	v.SetDefault("server.port", ":8088")
	v.SetDefault("auth.enable", false)
	v.SetDefault("db_path", "data.db")

	v.SetEnvPrefix("SEALDICE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Error reading config file: %v", err)
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	// Manual fallback for DBPath if not in structure
	c.DBPath = v.GetString("db_path")
	if c.DBPath == "" {
		c.DBPath = "data.db"
	}

	return &c
}
