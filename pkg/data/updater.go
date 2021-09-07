package data

import (
	"context"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/config"
	"gitlab.com/mswkn/bot/pkg/instrumenting"
	"go.uber.org/ratelimit"
	"os"
	"time"
)

type Updater struct {
	repo           mswkn.SecurityRepository
	conf           config.Config
	bulkUpdateSize int
	limiter        ratelimit.Limiter
}

func NewUpdater(conf config.Config, repo mswkn.SecurityRepository, bulkUpdateSize int) *Updater {
	u := &Updater{
		conf:           conf,
		repo:           repo,
		bulkUpdateSize: bulkUpdateSize,
		limiter:        ratelimit.New(1, ratelimit.Per(15*time.Minute)),
	}
	return u
}

func (u *Updater) StartUpdater(ctx context.Context) {
	lg := log.With().Str("comp", "updater").Logger()

	u.runUpdate(ctx, lg)

	cronChan := make(chan struct{})
	c := cron.New()
	addCron("1/15 7-23 * * 1-5", c, cronChan)
	c.Start()
	defer c.Stop()

	lg.Debug().Msg("starting xetra updater")
	for {
		select {
		case <-cronChan:
			u.runUpdate(ctx, lg)
		case <-ctx.Done():
			lg.Info().Msg("stopping xetra updater")
			return
		}
	}
}

func addCron(job string, c *cron.Cron, cronChan chan struct{}) {
	err := c.AddFunc(job, func() {
		cronChan <- struct{}{}
	})
	if err != nil {
		log.Fatal().Err(err).Str("comp", "updater").Str("cron", job).Msg("could not add cron job")
	}
}

func (u *Updater) runUpdate(ctx context.Context, lg zerolog.Logger) {
	u.limiter.Take()

	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	status := &instrumenting.DataUpdateStatus{
		LastUpdated: time.Now(),
	}

	if err := u.UpdateXetraFiles(ctx); err != nil {
		instrumenting.Status.SetDataUpdateStatus(status)
		lg.Error().Err(err).Msg("could not update xetra data")
		return
	}

	lg.Info().Msg("updated xetra data")
	instrumenting.Status.SetDataUpdateStatus(status)
}

func (u *Updater) UpdateXetraFiles(ctx context.Context) error {
	lg := log.With().Str("comp", "updater").Logger()
	lg.Info().Msg("start updating xetra data")

	secChan := make(chan *mswkn.Security)
	loaderErr := make(chan error)

	go func() {
		if u.conf.Data.XetraCSV == "" {
			lg.Info().Str("type", "online").Msg("locading xetra data")
			err := LoadXetraCSV(ctx, secChan)
			if err != nil {
				//ToDo send a reddit message to bot owner on errors
				loaderErr <- err
			}
		} else {
			lg.Info().Str("type", "local_file").Msg("locading xetra data")
			f, err := os.Open(u.conf.Data.XetraCSV)
			if err != nil {
				loaderErr <- err
				return
			}
			defer f.Close()
			if err := ParseXetraCSV(f, secChan); err != nil {
				loaderErr <- err
			}
		}
	}()

	if u.bulkUpdateSize == 1 {
		if err := u.add(ctx, loaderErr, secChan); err != nil {
			return err
		}
	} else {
		if err := u.addBulk(ctx, loaderErr, secChan); err != nil {
			return err
		}
	}

	return nil
}

func (u *Updater) add(ctx context.Context, loaderErr chan error, secChan chan *mswkn.Security) error {
	lg := log.With().Str("comp", "updater").Logger()

	for {
		select {
		case err := <-loaderErr:
			return err
		case s, ok := <-secChan:
			if !ok {
				log.Print("xetra update chan closed")
				return nil
			}

			if err := u.repo.Add(ctx, s); err != nil {
				lg.Error().
					Err(err).
					Str("isin", s.ISIN).
					Str("name", s.Name).
					Str("wkn", s.WKN).
					Msg("could not add security")
			} else if zerolog.GlobalLevel() == zerolog.TraceLevel {
				lg.Trace().Str("isin", s.ISIN).Msg("added security")
			}
		case <-ctx.Done():
			lg.Info().Msg("interrupted updating")
			return nil
		}
	}
}

func (u *Updater) addBulk(ctx context.Context, loaderErr chan error, secChan chan *mswkn.Security) error {
	lg := log.With().Str("comp", "updater").Logger()

	i := 0
	bulk := u.bulkUpdateSize
	list := make([]*mswkn.Security, 0, bulk)
	for {
		select {
		case err := <-loaderErr:
			return err
		case s, ok := <-secChan:
			if !ok {
				lg.Info().Msg("xetra bulk update data fully loaded")
				if err := u.repo.AddBulk(ctx, list); err != nil {
					lg.Error().Err(err).Msg("could not bulk add list")
				}
				lg.Trace().
					Int("bulk_count", len(list)).
					Int("i", i).
					Msg("added xetra securities")
				return nil
			}

			if len(list) < bulk {
				list = append(list, s)
				i++
				continue
			}

			if len(list) == bulk {
				if err := u.repo.AddBulk(ctx, list); err != nil {
					lg.Error().Err(err).Msg("could not add list")
				}

				lg.Trace().
					Int("bulk_count", len(list)).
					Int("i", i).
					Msg("added xetra securities")

				list = make([]*mswkn.Security, 0, bulk)
			}
		case <-ctx.Done():
			lg.Info().Msg("interrupted bulk updating")
			return nil
		}
	}
}
