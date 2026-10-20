package config

type Client struct {
	Slack Slack `mapstructure:"slack"`
}

type Slack struct {
	BotToken string `mapstructure:"bot_token"`
}
