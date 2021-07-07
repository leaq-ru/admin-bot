package config

type Config struct {
	HTTP        http
	STAN        stan
	NATS        nats
	Telegram    telegram
	Service     service
	LogLevel    string `envconfig:"LOGLEVEL"`
	ServiceName string
}

type http struct {
	Port string `envconfig:"HTTP_PORT"`
}

type stan struct {
	ClusterID               string `envconfig:"STAN_CLUSTERID"`
	SubjectReviewModeration string `envconfig:"STAN_SUBJECTREVIEWMODERATION"`
}

type nats struct {
	URL string `envconfig:"NATS_URL"`
}

type telegram struct {
	AdminBotWebhookURL string `envconfig:"TELEGRAM_ADMINBOTWEBHOOKURL"`
	AdminBotToken      string `envconfig:"TELEGRAM_ADMINBOTTOKEN"`
	AdminUserID        string `envconfig:"TELEGRAM_ADMINUSERID"`
}

type service struct {
	Parser string `envconfig:"SERVICE_PARSER"`
}
