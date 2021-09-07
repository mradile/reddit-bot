package responder

import (
	"bytes"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/reddit"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"os"
	"strings"
	"text/template"
	"time"
)

var tDefault = template.Must(template.New("default").Parse(replyTmpl))

var dePrinter = message.NewPrinter(language.German)

type Responder struct {
	client *reddit.Client
	msg    mswkn.Broker
}

func NewResponder(msg mswkn.Broker, client *reddit.Client) *Responder {
	r := &Responder{
		client: client,
		msg:    msg,
	}
	return r
}

func (s *Responder) Start(ctx context.Context) {
	lg := log.With().Str("comp", "responder").Logger()

	handler := func(rrr *mswkn.RedditReplyRequest) {
		lg := lg.With().Str("name", rrr.Name).Logger()
		lg.Debug().Msgf("received RedditReplyRequest: %+v", rrr)

		body, err := renderReply(rrr)
		if err != nil {
			lg.Error().Err(err).Msg("could not render response")
		}

		if os.Getenv("RESPOND_DRY_MODE") == "1" {
			log.Info().Str("respond", "RESPOND_DRY_MODE").Msg(body)
			return
		}

		lg.Trace().Str("body", body).Msg("rendered response")

		if os.Getenv("MSWKN_TEMPLATE_DEBUG") == "1" {
			fmt.Println(body)
		}

		lg.Trace().Msg("sending reddit reply")
		if err := s.client.Reply(rrr.Name, body); err != nil {
			lg.Error().Err(err).Msg("could not send response")
			return
		}
		lg.Info().Msg("reddit reply sent")
	}

	err := s.msg.Subscribe(mswkn.BrokerSubjectRedditRepplyRequest, handler)
	if err != nil {
		lg.Fatal().Err(err).Str("subject", mswkn.BrokerSubjectRedditRepplyRequest).Msg("could not subscribe to subject")
	}

	<-ctx.Done()
}

func renderReply(rrr *mswkn.RedditReplyRequest) (string, error) {
	replies := getReplyLines(rrr)
	buf := &bytes.Buffer{}
	err := tDefault.Execute(buf, replies)
	if err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}
	return buf.String(), nil
}

const replyTmpl = `
**WKNs:**

{{range . -}} 
{{.SecURL}} - {{.Name}}


{{end}}


**Details:**

|**WKN**|**Name**|**Type**|**Strike**|**Expire**|**Underlying**|
|:-|:-|:-|-:|:-|:-|:-|:-|
{{range . -}} 
|{{.SecURL}}|{{.Name}}|{{.Type}}|{{.Strike}}|{{.Expire}}|{{.Underlying}}|
{{end}}

^(ich bin ein bot)
`

type ReplyLine struct {
	SecURL     string
	Name       string
	Type       string
	Strike     string
	Expire     string
	Underlying string
}

func getReplyLines(rrr *mswkn.RedditReplyRequest) []*ReplyLine {
	replies := make([]*ReplyLine, 0)
	for _, wkn := range rrr.WKNs {
		sec, secFound := rrr.Securities[wkn]
		il, ilFound := rrr.InfoLinks[wkn]

		if !secFound {
			sec = &mswkn.Security{
				Name: "nix gefunden",
				WKN:  wkn,
			}
		}

		if !ilFound {
			il = &mswkn.InfoLink{
				WKN: wkn,
			}
		}

		rl := buildReplyLine(sec, il)

		if strings.TrimSpace(sec.Underlying) != "" {
			var underlying string
			if secU, ok := rrr.Securities[sec.Underlying]; ok {
				//rl.UnderlyingName = secU.Name
				underlying = fmt.Sprintf("%s %s", secU.Name, secU.WKN)
			} else {
				underlying = sec.Underlying
			}
			if ilU, ok := rrr.InfoLinks[sec.Underlying]; ok {
				underlying = infoLinkURL(underlying, ilU.URL)
			}

			rl.Underlying = underlying
		}

		replies = append(replies, rl)
	}
	return replies
}

func buildReplyLine(sec *mswkn.Security, il *mswkn.InfoLink) *ReplyLine {
	rl := &ReplyLine{
		SecURL:     infoLinkURL(il.WKN, il.URL),
		Name:       sec.Name,
		Type:       getTypeText(sec),
		Underlying: sec.Underlying,
	}

	if sec.Strike > 0 {
		rl.Strike = dePrinter.Sprintf("%.2f", sec.Strike)
	}

	if sec.Expire != nil {
		future := time.Now().Add(time.Hour * 24 * 365 * 20)

		if sec.Expire.Unix() < future.Unix() {
			rl.Expire = sec.Expire.Format("2006-01-02")
		}
	}

	return rl
}

func infoLinkURL(text, url string) string {
	if url != "" {
		return fmt.Sprintf("[%s](%s)", text, url)
	}
	return text
}

func getTypeText(sec *mswkn.Security) string {
	if sec.Type == mswkn.SecurityTypeWarrant {
		return fmt.Sprintf("%s %s",
			getWarrantSubType(sec.WarrantSubType),
			getWarrantType(sec.WarrantType),
		)
	}

	return getType(sec.Type)
}

func getWarrantSubType(wType int) string {
	switch wType {
	case mswkn.SecurityWarrantSubTypeKnockout:
		return "KO"
	case mswkn.SecurityWarrantSubTypeOS:
		return "OS"
	}
	return "-"
}

func getWarrantType(wType int) string {
	switch wType {
	case mswkn.SecurityWarrantTypeCall:
		return "Call"
	case mswkn.SecurityWarrantTypePut:
		return "Put"
	}
	return "-"
}

func getType(sType int) string {
	switch sType {
	case mswkn.SecurityTypeOption:
		return "Option"
	case mswkn.SecurityTypeFuture:
		return "Future"
	case mswkn.SecurityTypeBond:
		return "Bond"
	case mswkn.SecurityTypeCommonStock:
		return "Stock"
	case mswkn.SecurityTypeExchangeTradedFund:
		return "ETF"
	case mswkn.SecurityTypeExchangeTradedCommodity:
		return "ETC"
	case mswkn.SecurityTypeWarrant:
		return "Warrant"
	case mswkn.SecurityTypeExchangeTradedNode:
		return "ETN"
	}
	return "-"
}
