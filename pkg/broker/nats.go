package broker

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/config"
	"time"
)

type NatsClient struct {
	nc     *nats.Conn
	client *nats.EncodedConn
}

func NewNatsClient(conf config.Config, cancel context.CancelFunc) mswkn.Broker {
	lg := log.With().Str("comp", "broker").Logger()

	opts := make([]nats.Option, 0)
	opts = append(opts, nats.UserInfo(conf.Queue.Nats.Username, conf.Queue.Nats.Password))
	opts = append(opts, nats.RetryOnFailedConnect(true))
	opts = append(opts, nats.MaxReconnects(30))
	opts = append(opts, nats.ReconnectWait(time.Millisecond*250))
	opts = append(opts, nats.ErrorHandler(func(nc *nats.Conn, s *nats.Subscription, err error) {
		if s != nil {
			lg.Error().Msgf("Async error in %q/%q: %v", s.Subject, s.Queue, err)
		} else {
			lg.Error().Msgf("Async error outside subscription: %v", err)
		}
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		lg.Error().Msg("connection closed")
		cancel()
	}))

	connURL := fmt.Sprintf("%s:%d", conf.Queue.Nats.Host, conf.Queue.Nats.Port)

	nc, err := nats.Connect(connURL, opts...)
	if err != nil {
		lg.Fatal().Err(err).Msg("could not initialize nats connection")
	}

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		lg.Fatal().Err(err).Msg("could not create json connection")
	}

	n := &NatsClient{
		nc:     nc,
		client: ec,
	}

	return n
}

func (n *NatsClient) Publish(subject string, v interface{}) error {
	return n.client.Publish(subject, v)
}

func (n *NatsClient) Subscribe(subject string, handler interface{}) error {
	_, err := n.client.Subscribe(subject, handler)
	return err
}

func (n *NatsClient) Close() {
	n.client.Close()
	n.nc.Close()
}
