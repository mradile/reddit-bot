package securities

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"time"
)

type Securities struct {
	msg  mswkn.Broker
	repo mswkn.SecurityRepository
}

func NewService(msg mswkn.Broker, securities mswkn.SecurityRepository) *Securities {
	s := &Securities{
		msg:  msg,
		repo: securities,
	}
	return s
}

func (s *Securities) Start(ctx context.Context) {
	lg := log.With().Str("comp", "securities").Logger()

	handler := func(sr *mswkn.SecuritiesRequest) {
		lg := lg.With().Str("name", sr.Name).Logger()
		lg.Debug().Msgf("received SecuritiesRequest: %+v", sr)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		secs, underlyings := s.fetchSecurities(ctx, lg, sr.WKNs)
		if len(underlyings) > 0 {
			secsU, _ := s.fetchSecurities(ctx, lg, underlyings)
			for wkn, security := range secsU {
				secs[wkn] = security
			}
		}

		lg.Debug().Int("securities", len(secs)).Msg("wkn lookup done")

		ilf := &mswkn.InfoLinksRequest{
			Name:       sr.Name,
			WKNs:       sr.WKNs,
			Securities: secs,
		}
		lg.Trace().Msg("sending InfoLinksRequest")
		if err := s.msg.Publish(mswkn.BrokerSubjectInfoLinksRequest, ilf); err != nil {
			lg.Error().Err(err).Msg("could not send InfoLinksRequest")
			return
		}
		lg.Trace().Msg("InfoLinksRequest sent")
	}

	err := s.msg.Subscribe(mswkn.BrokerSubjectSecuritiesRequest, handler)
	if err != nil {
		lg.Fatal().Err(err).Str("subject", mswkn.BrokerSubjectSecuritiesRequest).Msg("could not subscribe to subject")
	}

	<-ctx.Done()
}
func (s *Securities) fetchSecurities(ctx context.Context, lg zerolog.Logger, wkns []string) (map[string]*mswkn.Security, []string) {
	secs := make(map[string]*mswkn.Security)
	underlyings := make([]string, 0)
	for _, wkn := range wkns {
		wkLg := lg.With().Str("wkn", wkn).Logger()

		sec, err := s.repo.Get(ctx, wkn)
		if err != nil {
			if err == mswkn.ErrSecurityNotFound {
				wkLg.Info().Msg("not found in repo")
			} else {
				wkLg.Error().Err(err).Msg("could not get security for wkn")
			}
			continue
		}
		secs[wkn] = sec
		wkLg.Debug().Msg("security found")

		if sec.Underlying != "" {
			wkLg.Debug().Str("underlying", sec.Underlying).Msg("underlying found")
			underlyings = append(underlyings, sec.Underlying)
		}
	}
	return secs, underlyings
}

/*
for _, wkn := range wr.WKNs {
			wkLg := lg.With().Str("wkn", wkn).Logger()
			sec, err := s.repo.Get(ctx, wkn)
			if err != nil {
				if err == mswkn.ErrSecurityNotFound {
					wkLg.Info().Msg("not found in repo")
					errors[wkn] = "kein Wertpapier gefunden"
					continue
				}
				errors[wkn] = "internal error"
				wkLg.Error().Err(err).Msg("could not get security for wkn")
				continue
			}
			secs[wkn] = sec
			wkLg.Debug().Msg("security found")

			if sec.Underlying != "" {
				wkLg.Debug().Str("underlying", sec.Underlying).Msg("underlying found")
				//tmpWkn = append(tmpWkn, sec.Underlying)
			}
		}
*/
