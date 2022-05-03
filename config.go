package main

import (
	"path"

	"github.com/spf13/viper"
)

type Config struct {
	Interval        int    `mapstructure:"interval"`
	SlackWebhookUrl string `mapstructure:"slack_webhook_url"`
	DocsPath        string `mapstructure:"docs_path"`
	Docs            []Doc  `mapstructure:"docs"`
}
type Doc struct {
	Name string `mapstructure:"name"`
	Url  string `mapstructure:"url"`
}

func (d *Doc) Path() string {
	return path.Join(config.DocsPath, d.Name)
}

func initializeConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	viper.Unmarshal(&config)
}
