package scanner

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"regexp"
	"strings"
)

//BotKeyWord is the keyword to trigger the bot
const BotKeyWord = "$WKN"

var (
	//ErrEmptyBodyText error returned when the text body has a length of 0
	ErrEmptyBodyText = errors.New("body is empty")
	//ErrNoTokens error returned when token created from the text body are <=1
	ErrNoTokens = errors.New("no tokens found in body")
)

type Scanner struct {
	msg mswkn.Broker
}

func NewScanner(msg mswkn.Broker) *Scanner {
	s := &Scanner{
		msg: msg,
	}
	return s
}

func (s *Scanner) Start(ctx context.Context) {
	lg := log.With().Str("comp", "scanner").Logger()

	handler := func(wr *mswkn.RedditRequest) {
		lg := lg.With().Str("name", wr.Name).Logger()
		lg.Debug().Msgf("received RedditRequest: %+v", wr)

		wkns, err := DollarWKNTokenScan(wr.Text)
		if err != nil {
			if err == ErrEmptyBodyText {
				lg.Debug().Msg("body is empty")
			} else if err == ErrNoTokens {
				lg.Debug().Msg("body contains no tokens")
			} else {
				lg.Error().Err(err).Msg("could not scan body for comment")
			}
			return
		}

		if len(wkns) < 1 {
			lg.Debug().Msg("ignoring comment because it has no tokens")
			return
		}

		sr := &mswkn.SecuritiesRequest{
			Name: wr.Name,
			WKNs: wkns,
		}
		lg.Trace().Msg("sending SecuritiesRequest")
		if err := s.msg.Publish(mswkn.BrokerSubjectSecuritiesRequest, sr); err != nil {
			lg.Error().Err(err).Str("name", wr.Name).Msg("could not send SecuritiesRequest")
			return
		}
		lg.Trace().Msg("SecuritiesRequest sent")
	}

	err := s.msg.Subscribe(mswkn.BrokerSubjectWKNRequest, handler)
	if err != nil {
		lg.Fatal().Err(err).Str("subject", mswkn.BrokerSubjectWKNRequest).Msg("could not subscribe to subject")
	}

	<-ctx.Done()
}

func DummyScanner(text string) ([]string, error) {
	return []string{"TT6DHP"}, nil
}

func DollarWKNTokenScan(text string) ([]string, error) {
	if text == "" {
		return nil, ErrEmptyBodyText
	}

	text = strings.ToUpper(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "  ", " ")
	text = strings.ReplaceAll(text, "  ", " ")

	tokens := strings.Split(text, " ")

	wknMap := make(map[string]bool)
	if len(tokens) > 1 {
		dollarWKNKeywordTokenScan(tokens, wknMap)
	}
	dollarWKNTokenScan(tokens, wknMap)

	if len(wknMap) == 0 {
		return nil, nil
	}

	wkns := make([]string, 0, len(wknMap))
	for wkn := range wknMap {
		wkns = append(wkns, wkn)
	}
	return wkns, nil
}

var dollarWKNTokenRegEx = regexp.MustCompile(`\$([A-Z0-9]{6})`)

func dollarWKNTokenScan(tokens []string, wknMap map[string]bool) {
	for _, token := range tokens {
		token = strings.TrimSpace(token)

		//$AA88YY
		if len(token) != 7 {
			continue
		}

		findings := dollarWKNTokenRegEx.FindStringSubmatch(token)
		for _, finding := range findings {
			if strings.HasPrefix(finding, "$") {
				continue
			}
			wknMap[finding] = true
		}
	}
}

func dollarWKNKeywordTokenScan(tokens []string, wknMap map[string]bool) {
	skipNext := false
	tCount := len(tokens)
	for i, token := range tokens {
		if skipNext {
			skipNext = false
			continue
		}

		if token == BotKeyWord {
			wknI := i + 1
			if wknI == tCount {
				continue
			}
			wkn := tokens[wknI]
			if len(wkn) == 6 {
				wknMap[wkn] = true
				skipNext = true
			}
		}
	}
}

var wknTokenRegEx = regexp.MustCompile(`([A-Z0-9]{6})`)

func WKNDollarFullScan(text string) ([]string, error) {
	if text == "" {
		return nil, ErrEmptyBodyText
	}

	text = strings.ToUpper(text)

	inTokens := strings.Split(text, " ")
	matches := make(map[string]bool)
	for _, token := range inTokens {
		token = strings.TrimSpace(token)

		if len(token) != 6 {
			continue
		}

		findings := wknTokenRegEx.FindStringSubmatch(token)
		for _, finding := range findings {
			matches[finding] = true
		}
	}

	if len(matches) < 1 {
		return nil, nil
	}

	wkns := make([]string, 0)
	for wkn := range matches {
		wkns = append(wkns, wkn)
	}

	return wkns, nil
}
