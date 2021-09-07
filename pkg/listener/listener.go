package listener

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/turnage/graw/reddit"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/config"
	r "gitlab.com/mswkn/bot/pkg/reddit"
)

type commentListener struct {
	ignoreUser string
	msg        mswkn.Broker
}

func (c *commentListener) Comment(post *reddit.Comment) error {
	lg := log.With().Str("comp", "listener").Str("name", post.Name).Logger()
	lg.Debug().Msgf("received reddit comment: %s", post.Body)

	if post.Author == c.ignoreUser {
		lg.Debug().Msg("got own comment")
		return nil
	}

	rc := &mswkn.RedditRequest{
		Name: post.Name,
		Text: post.Body,
	}
	lg.Trace().Msg("sending RedditRequest")
	if err := c.msg.Publish(mswkn.BrokerSubjectWKNRequest, rc); err != nil {
		lg.Error().Err(err).Msg("could not send RedditRequest")
		return nil
	}
	lg.Trace().Msg("RedditRequest sent")
	return nil
}

type Listener struct {
	client *r.Client
	conf   config.Config
	msg    mswkn.Broker
}

func NewListener(conf config.Config, client *r.Client, msg mswkn.Broker) *Listener {
	l := &Listener{
		client: client,
		conf:   conf,
		msg:    msg,
	}
	return l
}

func (l *Listener) Start(ctx context.Context) {
	p := r.CommentHandlerParams{
		Ctx: ctx,
		Handler: &commentListener{
			ignoreUser: l.conf.Reddit.Username,
			msg:        l.msg,
		},
		SubReddits: l.conf.Reddit.SubReddits,
	}

	start := l.client.RegisterCommentHandler(p)

	if err := start(); err != nil {
		log.Fatal().Err(err).Str("comp", "listener").Msg("could not start reddit listener")
	}
}
