package onvista

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"go.uber.org/ratelimit"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://www.onvista.de"
const baseURLCommonStock = baseURL + "/aktien/"
const baseURLWarrant = baseURL + "/derivate/snapshot"
const baseURLETF = baseURL + "/fonds/snapshot"

type Client struct {
	c       *http.Client
	limiter ratelimit.Limiter
}

func NewClient() *Client {
	c := &Client{
		c: &http.Client{
			Timeout: time.Second * 15,
			//CheckRedirect: func(req *http.RedditRequest, via []*http.RedditRequest) error {
			//	return http.ErrUseLastResponse
			//},
		},
		limiter: ratelimit.New(60, ratelimit.Per(60*time.Second)),
	}
	return c
}

// ToDo use onvista search API 'https://www.onvista.de/onvista/boxes/assetSearch.json?doSubmit=Suchen&portfolioName=&searchValue=TT6DHP'
func (c *Client) FetchURLByWKN(ctx context.Context, wkn string) (string, error) {
	panic("implement me")
}

func (c *Client) FetchURL(ctx context.Context, sec *mswkn.Security) (string, error) {
	var reqURL string
	switch sec.Type {
	case mswkn.SecurityTypeCommonStock:
		reqURL = baseURLCommonStock
	case mswkn.SecurityTypeWarrant:
		reqURL = baseURLWarrant
	case mswkn.SecurityTypeExchangeTradedNode:
		reqURL = baseURLWarrant
	case mswkn.SecurityTypeExchangeTradedFund:
		reqURL = baseURLETF
	default:
		log.Info().Str("wkn", sec.WKN).Int("secType", sec.Type).Msg("using default baseURL in onvista client")
		reqURL = baseURL + "/"
	}
	reqURL = fmt.Sprintf("%s/%s", reqURL, strings.ToUpper(sec.ISIN))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}

	c.limiter.Take()

	res, err := c.c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	log.Debug().Str("loc", loc).Str("url", reqURL).Msg("fetched from onvista")
	if loc == "" && res.Request != nil {
		loc = res.Request.URL.String()
	}
	if loc == "" {
		return "", errors.New("could not guess an URL")
	}

	return loc, nil

	//switch sec.Type {
	//case mswkn.SecurityTypeCommonStock:
	//	il.URL = loc
	//case mswkn.SecurityTypeWarrant:
	//	il.URL = fmt.Sprintf("%s%s", baseURL, loc)
	//}
}
