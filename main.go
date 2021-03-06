package main

import (
	"context"
	"github.com/leaq-ru/admin-bot/bot"
	"github.com/leaq-ru/admin-bot/call"
	"github.com/leaq-ru/admin-bot/config"
	"github.com/leaq-ru/admin-bot/healthz"
	"github.com/leaq-ru/admin-bot/logger"
	"github.com/leaq-ru/admin-bot/stan"
	graceful "github.com/leaq-ru/lib-graceful"
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

	companyClient, reviewClient, err := call.NewClients(cfg.Service.Parser)
	logg.Must(err)

	b, err := bot.NewBot(
		logg.ZL,
		cfg.HTTP.Port,
		cfg.Telegram.AdminBotWebhookURL,
		cfg.Telegram.AdminBotToken,
		cfg.Telegram.AdminUserID,
		companyClient,
		reviewClient,
	)
	logg.Must(err)

	sc, err := stan.NewConn(cfg.ServiceName, cfg.STAN.ClusterID, cfg.NATS.URL)
	logg.Must(err)

	cons, err := stan.NewConsumer(
		logg.ZL,
		sc,
		cfg.STAN.SubjectReviewModeration,
		cfg.ServiceName,
		0,
		b.ReviewModeration,
	)
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
