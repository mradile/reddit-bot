package reddit

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/turnage/graw"
	"github.com/turnage/graw/reddit"
	"gitlab.com/mswkn/bot/pkg/config"
	"net/http"
	"time"
)

type Client struct {
	bot reddit.Bot
}

func NewClient(conf config.Config) *Client {
	bCfg := reddit.BotConfig{
		Agent: conf.Reddit.Agent,
		App: reddit.App{
			ID:       conf.Reddit.ClientID,
			Secret:   conf.Reddit.ClientSecret,
			Username: conf.Reddit.Username,
			Password: conf.Reddit.Password,
		},
		Client: &http.Client{
			Timeout: time.Second * 15,
		},
	}
	bot, err := reddit.NewBot(bCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize bot")
	}

	c := &Client{bot: bot}
	return c
}

type CommentHandlerParams struct {
	Ctx        context.Context
	Handler    interface{}
	SubReddits []string
}

func (c *Client) Reply(name, text string) error {
	lg := log.With().Str("comp", "reddit").Str("name", name).Logger()

	lg.Debug().Msg("trying to sent comment")
	if err := c.bot.Reply(name, text); err != nil {
		return err
	}
	lg.Debug().Msg("comment sent")

	return nil

}

func (c *Client) RegisterCommentHandler(p CommentHandlerParams) func() error {
	cfg := graw.Config{
		SubredditComments: p.SubReddits,
	}

	stop, wait, err := graw.Run(p.Handler, c.bot, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("could not register reddit listener")
	}

	go func() {
		<-p.Ctx.Done()
		stop()
	}()

	return wait

}
