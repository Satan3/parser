package main

type Config struct {
	DatabaseURL          string `toml:"database_url"`
	GoroutinesMultiplier int    `toml:"goroutines_multiplier"`
	SendTo               string `toml:"database"`
	TelegramBotKey       string `toml:"telegram_bot_key"`
}

func NewConfig() *Config {
	return &Config{}
}
