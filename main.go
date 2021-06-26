package main

import (
	"github.com/nnqq/scr-admin-bot/bot"
	"github.com/nnqq/scr-admin-bot/call"
	"github.com/nnqq/scr-admin-bot/config"
	"github.com/nnqq/scr-admin-bot/healthz"
	"github.com/nnqq/scr-admin-bot/logger"
	graceful "github.com/nnqq/scr-lib-graceful"
	"log"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	logg, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	companyClient, err := call.NewClients(cfg.Service.Parser)
	logg.Must(err)

	b, err := bot.NewBot(
		logg.ZL,
		cfg.HTTP.Port,
		cfg.Telegram.AdminBotWebhookURL,
		cfg.Telegram.AdminBotToken,
		cfg.Telegram.AdminUserID,
		companyClient,
	)
	logg.Must(err)

	go healthz.Start(81)
	go graceful.HandleSignals(b.Stop)
	b.Start()
}
