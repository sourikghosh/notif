package config

import (
	"time"

	"github.com/spf13/viper"
)

type NotifConfig struct {
	PORT              string `mapstructure:"PORT"`
	Mode              string `mapstructure:"MODE"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`
	Encoding          string `mapstructure:"ENCODING"`
	EmailSmtpPassword string `mapstructure:"EMAIL_SMTP_PASSWORD"`
	EmailSmtpUserName string `mapstructure:"EMAIL_SMTP_USERNAME"`
	EmailSmtpHost     string `mapstructure:"EMAIL_SMTP_HOST"`
	EmailSmtpPORT     string `mapstructure:"EMAIL_SMTP_PORT"`
	Emailtest         string `mapstructure:"EMAIL_TO_SEND"`
}

var defaultsValue = map[string]string{
	"PORT":            "6969",
	"MODE":            Development,
	"EMAIL_SMTP_HOST": "smtp.gmail.com",
	"EMAIL_SMTP_PORT": "587",
}

func LoadConfig(path string) (*NotifConfig, error) {
	// "" -> loads timezone as UTC:
	loc, err := time.LoadLocation("")
	if err != nil {
		return nil, err
	}

	time.Local = loc
	//Checks the defaults Value map and sets the default
	for key, val := range defaultsValue {
		viper.SetDefault(key, val)
	}

	viper.AddConfigPath(path)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var cfg NotifConfig

	err = viper.Unmarshal(&cfg)

	return &cfg, err
}
