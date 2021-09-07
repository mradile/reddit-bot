package main

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot/pkg"
	"gitlab.com/mswkn/bot/pkg/config"
	"os"
	"os/signal"
	"strings"
	"time"
)

var version = "snapshot"

func main() {
	conf := config.LoadConfigFromEnv(version)

	setLogLevel(conf)

	ctx, cancel := handleSignals()
	app := pkg.NewApp(conf)
	if err := app.Start(ctx, cancel); err != nil {
		log.Fatal().Err(err).Msg("could not start bot")
	}
}

func setLogLevel(conf config.Config) {
	switch strings.ToLower(conf.LogLevel) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	if conf.LogMode == "develop" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}

}

func handleSignals() (context.Context, context.CancelFunc) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	cancel := func() {
		signal.Stop(c)
		ctxCancel()
	}
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
