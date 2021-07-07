package main

import (
	"context"
	"github.com/nnqq/scr-admin-bot/bot"
	"github.com/nnqq/scr-admin-bot/call"
	"github.com/nnqq/scr-admin-bot/config"
	"github.com/nnqq/scr-admin-bot/consumer"
	"github.com/nnqq/scr-admin-bot/healthz"
	"github.com/nnqq/scr-admin-bot/logger"
	"github.com/nnqq/scr-admin-bot/stan"
	graceful "github.com/nnqq/scr-lib-graceful"
	"log"
	"sync"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

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

	sc, err := stan.NewConn(cfg.ServiceName, cfg.STAN.ClusterID, cfg.NATS.URL)
	logg.Must(err)

	cons, err := consumer.NewConsumer(logg.ZL, sc, cfg.STAN.SubjectReviewModeration, b.ReviewModeration)
	logg.Must(err)

	go healthz.Start(81)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		graceful.HandleSignals(cancel)
	}()
	go func() {
		defer wg.Done()
		b.Serve(ctx)
	}()
	go func() {
		defer wg.Done()
		cons.Serve(ctx)
	}()
	wg.Wait()
}
