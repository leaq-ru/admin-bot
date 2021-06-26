package bot

import (
	"context"
	"fmt"
	"github.com/nnqq/scr-proto/codegen/go/parser"
	"github.com/rs/zerolog"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type bot struct {
	bot           *tb.Bot
	adminUserID   int
	companyClient parser.CompanyClient
	logger        zerolog.Logger
}

func NewBot(logger zerolog.Logger, httpPort, webhookURL, token, rawAdminUserID string, companyClient parser.CompanyClient) (*bot, error) {
	adminUserID, err := strconv.Atoi(rawAdminUserID)
	if err != nil {
		return nil, err
	}

	publicURL := fmt.Sprintf("%s/%s", webhookURL, token)
	poller := &tb.Webhook{
		Listen: "0.0.0.0:" + httpPort,
		Endpoint: &tb.WebhookEndpoint{
			PublicURL: publicURL,
		},
	}
	onlyMe := tb.NewMiddlewarePoller(poller, func(upd *tb.Update) bool {
		if upd.Message != nil && upd.Message.Sender != nil && upd.Message.Sender.ID == adminUserID {
			return true
		}
		return false
	})

	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: onlyMe,
	})
	if err != nil {
		return nil, err
	}

	return &bot{
		bot:           b,
		adminUserID:   adminUserID,
		companyClient: companyClient,
		logger:        logger.With().Str("package", "bot").Logger(),
	}, nil
}

func (b *bot) initHandlers() {
	b.bot.Handle("/help", func(m *tb.Message) {
		b.reply(m, `Set company hidden
/h
https://leaq.ru/company/slug1
https://leaq.ru/company/slug2`)
	})

	b.bot.Handle("/h", func(m *tb.Message) {
		const (
			invalidURL = "error: invalid URL"
			newline    = "\n"
		)

		urls := strings.Split(m.Payload, newline)
		var slugs []string
		for _, u := range urls {
			switch u {
			case "", " ", newline:
				continue
			default:
				pu, err := url.Parse(u)
				if err != nil || pu.Path == "" {
					b.reply(m, invalidURL)
					return
				}

				slugs = append(slugs, strings.TrimPrefix(pu.Path, "company/"))
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := b.companyClient.SetHidden(ctx, &parser.SetHiddenRequest{
			Slugs: slugs,
		})
		if err != nil {
			b.logger.Error().Err(err)
			b.reply(m, "error: can't set hidden")
			return
		}

		b.reply(m, strings.Join(append(
			[]string{"OK"},
			slugs...,
		), newline))
	})
}

func (b *bot) reply(m *tb.Message, text string) bool {
	if m == nil {
		return false
	}

	_, err := b.bot.Send(m.Sender, text, &tb.SendOptions{
		ReplyTo: m,
	})
	if err != nil {
		b.logger.Error().Err(err)
		return false
	}
	return true
}

func (b *bot) Start() {
	b.initHandlers()
	b.bot.Start()
}

func (b *bot) Stop() {
	b.bot.Stop()
}
