package config

type Config struct {
	HTTP     http
	Telegram telegram
	Service  service
	LogLevel string `envconfig:"LOGLEVEL"`
}

type http struct {
	Port string `envconfig:"HTTP_PORT"`
}

type telegram struct {
	AdminBotWebhookURL string `envconfig:"TELEGRAM_ADMINBOTWEBHOOKURL"`
	AdminBotToken      string `envconfig:"TELEGRAM_ADMINBOTTOKEN"`
	AdminUserID        string `envconfig:"TELEGRAM_ADMINUSERID"`
}

type service struct {
	Parser string `envconfig:"SERVICE_PARSER"`
}
