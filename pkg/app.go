package pkg

import (
	"context"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/broker"
	"gitlab.com/mswkn/bot/pkg/config"
	"gitlab.com/mswkn/bot/pkg/data"
	"gitlab.com/mswkn/bot/pkg/db"
	"gitlab.com/mswkn/bot/pkg/http/rest"
	"gitlab.com/mswkn/bot/pkg/infolinks"
	"gitlab.com/mswkn/bot/pkg/listener"
	"gitlab.com/mswkn/bot/pkg/onvista"
	"gitlab.com/mswkn/bot/pkg/reddit"
	"gitlab.com/mswkn/bot/pkg/responder"
	"gitlab.com/mswkn/bot/pkg/scanner"
	"gitlab.com/mswkn/bot/pkg/securities"
	"time"
)

type App struct {
	conf config.Config
}

func NewApp(conf config.Config) *App {
	a := &App{
		conf: conf,
	}
	return a
}

func (a *App) Start(ctx context.Context, cancel context.CancelFunc) error {
	lg := log.With().Str("comp", "app").Logger()
	lg.Info().Str("version", a.conf.Version).Msg("initializing")

	msg := broker.NewNatsClient(a.conf, cancel)

	bulkUpdateSize := 100_000

	//initialize repositories
	var secRepo mswkn.SecurityRepository
	var infoLinkRepo mswkn.InfoLinkRepository
	if a.conf.Database.Pg.Enabled {
		pgDB := db.NewPgDb(a.conf)
		defer pgDB.Close()
		secRepo = db.NewPgSecurityRepository(pgDB)
		infoLinkRepo = db.NewPgInfoLinkRepository(pgDB)
		bulkUpdateSize = 5_000
		lg.Info().Msg("using postgres data backend")
	} else {
		secRepo = db.NewMemorySecurityRepository()
		infoLinkRepo = db.NewMemoryInfoLinkRepository()
		lg.Info().Msg("using memory data backend")
	}

	onvistaClient := onvista.NewClient()
	updater := data.NewUpdater(a.conf, secRepo, bulkUpdateSize)
	redditClient := reddit.NewClient(a.conf)
	commentListener := listener.NewListener(a.conf, redditClient, msg)
	secService := securities.NewService(msg, secRepo)
	infoLinkService := infolinks.NewService(msg, onvistaClient, infoLinkRepo)
	responderService := responder.NewResponder(msg, redditClient)

	httpServer := rest.NewServer(a.conf, msg, secRepo, nil)

	//	wg := sync.WaitGroup{}

	go func() {
		defer cancel()
		lg.Debug().Msg("starting data updater")
		updater.StartUpdater(ctx)
	}()

	go func() {
		defer cancel()
		lg.Debug().Msg("starting comment listener")
		commentListener.Start(ctx)
	}()

	go func() {
		defer cancel()
		lg.Debug().Msg("starting scanner")
		scanner.NewScanner(msg).Start(ctx)
	}()

	go func() {
		defer cancel()
		lg.Debug().Msg("starting security service")
		secService.Start(ctx)
	}()

	go func() {
		defer cancel()
		lg.Debug().Msg("starting infolink service")
		infoLinkService.Start(ctx)
	}()

	go func() {
		defer cancel()
		lg.Debug().Msg("starting responder service")
		responderService.Start(ctx)
	}()

	go func() {
		defer cancel()
		defer httpServer.Stop(context.Background())
		lg.Debug().Msg("starting http server")
		if err := httpServer.Start(); err != nil {
			lg.Error().Err(err).Msg("shutting down http server")
		}
	}()

	//wait for stop signal
	<-ctx.Done()
	lg.Info().Msg("bot is shutting down")

	msg.Close()

	httpCtx, httpCancel := context.WithTimeout(ctx, time.Second*5)
	defer httpCancel()
	if err := httpServer.Stop(httpCtx); err != nil {
		lg.Error().Err(err).Msg("")
	}

	return nil
}
