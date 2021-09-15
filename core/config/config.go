package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	viper.Viper
}

func NewConfig(confDir *string) (*Config, error) {
	v := viper.New()
	if v == nil {
		return nil, errors.New("Error initializing internal config")
	}
	if confDir != nil {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(*confDir)
		if err := v.ReadInConfig(); err != nil {
			return nil, errors.Wrapf(err, "problem reading config from [%s]", *confDir)
		}
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return &Config{
		*v,
	}, nil
}