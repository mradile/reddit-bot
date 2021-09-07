package infolinks

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/onvista"
	"time"
)

type InfoLinks struct {
	msg           mswkn.Broker
	ilRepo        mswkn.InfoLinkRepository
	onvistaClient *onvista.Client
}

func NewService(msg mswkn.Broker, onvistaClient *onvista.Client, ilRepo mswkn.InfoLinkRepository) *InfoLinks {
	s := &InfoLinks{
		msg:           msg,
		onvistaClient: onvistaClient,
		ilRepo:        ilRepo,
	}
	return s
}

func (i *InfoLinks) Start(ctx context.Context) {
	lg := log.With().Str("comp", "infolinks").Logger()

	handler := func(wr *mswkn.InfoLinksRequest) {
		lg := lg.With().Str("name", wr.Name).Logger()
		lg.Debug().Msgf("received InfoLinksRequest: %+v", wr)

		infoLinks := make(map[string]*mswkn.InfoLink)

		for wkn, sec := range wr.Securities {
			il, err := i.fetchInfoLink(sec)
			if err != nil {
				lg.Error().Err(err).Msg("could not fetch link")
				continue
			}
			infoLinks[wkn] = il
		}

		//ToDo iterate errors and try to fetch infolinks from FetchURLByWKN

		lg.Debug().Int("infolinks", len(infoLinks)).Msg("infolink lookup done")

		ilf := &mswkn.RedditReplyRequest{
			Name:       wr.Name,
			WKNs:       wr.WKNs,
			Securities: wr.Securities,
			InfoLinks:  infoLinks,
		}
		lg.Trace().Msg("sending RedditReplyRequest")
		if err := i.msg.Publish(mswkn.BrokerSubjectRedditRepplyRequest, ilf); err != nil {
			lg.Error().Err(err).Msg("could not send RedditReplyRequest")
			return
		}
		lg.Trace().Msg("RedditReplyRequest sent")
	}

	err := i.msg.Subscribe(mswkn.BrokerSubjectInfoLinksRequest, handler)
	if err != nil {
		lg.Fatal().Err(err).Str("subject", mswkn.BrokerSubjectInfoLinksRequest).Msg("could not subscribe to subject")
	}

	<-ctx.Done()
}

func (i *InfoLinks) fetchInfoLink(sec *mswkn.Security) (*mswkn.InfoLink, error) {
	lg := log.With().Str("comp", "infolinks").Str("wkn", sec.WKN).Logger()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	il := &mswkn.InfoLink{
		WKN: sec.WKN,
	}

	if cached, err := i.ilRepo.Get(ctx, il.WKN); err != nil {
		if err == mswkn.ErrInfoLinkNotFound {
			lg.Debug().Msg("cache miss")
		} else {
			lg.Error().Err(err).Msg("could not fetch a cached info link")
		}
	} else {
		lg.Debug().Msg("cache hit")
		return cached, nil
	}

	secURL, err := i.onvistaClient.FetchURL(ctx, sec)
	if err != nil {
		return nil, fmt.Errorf("could not fetch a link from onvista: %w", err)
	}

	il.URL = secURL

	lg.Debug().Str("url", il.URL).Msg("fetched url")

	if il.URL == "" {
		return nil, fmt.Errorf("url was empty from onvista: %w", err)
	}

	if err := i.ilRepo.Add(ctx, il); err != nil {
		lg.Error().Err(err).Msg("could not cache a link from onvista")
	}

	return il, nil
}
