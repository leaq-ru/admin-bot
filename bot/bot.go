package bot

import (
	"context"
	"errors"
	"fmt"
	"github.com/leaq-ru/proto/codegen/go/event"
	"github.com/leaq-ru/proto/codegen/go/parser"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type bot struct {
	bot           *tb.Bot
	adminUserID   int
	companyClient parser.CompanyClient
	reviewClient  parser.ReviewClient
	logger        zerolog.Logger
}

const newline = "\n"

const (
	newReview        = "📔 New review"
	prefixReviewID   = "ReviewID: "
	prefixUserID     = "UserID: "
	cmdReviewApprove = "+"
	cmdReviewDecline = "-"
	cmdReviewBan     = "ban"
)

func NewBot(
	logger zerolog.Logger,
	httpPort,
	webhookURL,
	token,
	rawAdminUserID string,
	companyClient parser.CompanyClient,
	reviewClient parser.ReviewClient,
) (
	*bot,
	error,
) {
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
		reviewClient:  reviewClient,
		logger:        logger.With().Str("package", "bot").Logger(),
	}, nil
}

func (b *bot) Serve(ctx context.Context) {
	b.initHandlers()
	go func() {
		<-ctx.Done()
		b.bot.Stop()
	}()
	b.logger.Debug().Msg("bot started")
	b.bot.Start()
	b.logger.Debug().Msg("bot stopped")
}

func (b *bot) ReviewModeration(rawMsg *stan.Msg) {
	var msg event.ReviewModeration
	err := protojson.Unmarshal(rawMsg.Data, &msg)
	if err != nil {
		b.logger.Error().Err(err).Send()
		return
	}

	ok := b.sendToAdmin(fmt.Sprintf(`%s

%s%s
%s%s
Text: %s`,
		newReview,
		prefixReviewID, msg.GetReview().GetId(),
		prefixUserID, msg.GetReview().GetUser().GetId(),
		msg.GetReview().GetText()))
	if ok {
		err = rawMsg.Ack()
		if err != nil {
			b.logger.Error().Err(err).Send()
		}
	}
}

func (b *bot) sendToAdmin(msg string) bool {
	_, err := b.bot.Send(&tb.User{ID: b.adminUserID}, msg)
	if err != nil {
		b.logger.Error().Err(err).Send()
		return false
	}
	return true
}

func (b *bot) initHandlers() {
	b.bot.Handle(tb.OnText, func(msg *tb.Message) {
		if !msg.IsReply() || !strings.HasPrefix(msg.ReplyTo.Text, newReview) {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		payload := strings.Split(msg.ReplyTo.Text, newline)
		reviewID := strings.TrimPrefix(payload[2], prefixReviewID)
		userID := strings.TrimPrefix(payload[3], prefixUserID)

		var err error
		switch msg.Text {
		case cmdReviewApprove:
			_, err = b.reviewClient.ChangeStatus(ctx, &parser.ChangeStatusRequest{
				ReviewId: reviewID,
				Status:   parser.ReviewStatus_OK,
			})
		case cmdReviewDecline:
			_, err = b.reviewClient.ChangeStatus(ctx, &parser.ChangeStatusRequest{
				ReviewId: reviewID,
				Status:   parser.ReviewStatus_DELETE,
			})
		case cmdReviewBan:
			_, err = b.reviewClient.DisallowAuthorDeleteAll(ctx, &parser.DisallowAuthorDeleteAllRequest{
				UserId: userID,
			})
		default:
			b.replyErr(msg, "Unknown reply-command")
			return
		}
		if err != nil {
			b.logger.Error().Err(err).Send()
			b.replyErr(msg, "Can't call rpc")
			return
		}

		b.replyOK(msg, reviewID)
	})

	b.bot.Handle("/help", func(msg *tb.Message) {
		b.reply(msg, fmt.Sprintf(`Set company hidden
/h
https://leaq.ru/company/slug1
https://leaq.ru/company/slug2

When got message '%s'
Reply '%s' to approve, '%s' to decline, '%s' to dissallow user reviews and delete all`,
			newReview,
			cmdReviewApprove, cmdReviewDecline, cmdReviewBan))
	})

	const hCmd = "/h"
	b.bot.Handle(hCmd, func(msg *tb.Message) {
		const invalidURL = "invalid URL"

		if msg.Text == "" {
			return
		}

		urls := filterCmd(hCmd, strings.Split(msg.Text, newline))
		var slugs []string
		for _, u := range urls {
			pu, err := url.Parse(u)
			if err != nil || pu.Path == "" {
				b.replyErr(msg, invalidURL)
				return
			}

			slugs = append(slugs, strings.TrimPrefix(pu.Path, "/company/"))
		}

		if len(slugs) == 0 {
			b.replyErr(msg, "No URLs provided")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := b.companyClient.SetHidden(ctx, &parser.SetHiddenRequest{
			Slugs: slugs,
		})
		if err != nil {
			b.logger.Error().Err(err)
			b.replyErr(msg, "Can't set hidden")
			return
		}

		var eg errgroup.Group
		for _, _u := range urls {
			u := _u
			eg.Go(func() error {
				req, e := http.NewRequest(http.MethodGet, u, nil)
				if e != nil {
					return e
				}
				req.Header.Set("User-Agent", "Bot")

				res, e := new(http.Client).Do(req)
				if e != nil {
					return e
				}

				if res.StatusCode == http.StatusNotFound {
					return nil
				}

				return errors.New("response not 404, URL=" + u)
			})
		}
		err = eg.Wait()
		if err != nil {
			b.logger.Error().Err(err)
			b.replyErr(msg, "URL post-hide check failed, seems urls response not 404")
			return
		}

		b.replyOK(msg, slugs...)
	})
}

func (b *bot) reply(msg *tb.Message, text string) bool {
	if msg == nil {
		return false
	}

	_, err := b.bot.Send(msg.Sender, text, &tb.SendOptions{
		ReplyTo: msg,
	})
	if err != nil {
		b.logger.Error().Err(err)
		return false
	}
	return true
}

func (b *bot) replyOK(msg *tb.Message, strs ...string) bool {
	return b.reply(msg, strings.Join(append([]string{"OK"}, strs...), newline))
}

func (b *bot) replyErr(msg *tb.Message, strs ...string) bool {
	return b.reply(msg, strings.Join(append([]string{"Error"}, strs...), newline))
}

func filterCmd(cmd string, text []string) []string {
	var res []string
	for _, s := range text {
		switch s {
		case "", " ", "\n", cmd:
			continue
		default:
			res = append(res, s)
		}
	}
	return res
}
