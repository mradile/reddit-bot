package responder

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/mswkn/bot"
	"testing"
)

func Test_getReplyLines(t *testing.T) {
	type args struct {
		rrr *mswkn.RedditReplyRequest
	}
	tests := []struct {
		name string
		args args
		want []*ReplyLine
	}{
		{
			name: "simple",
			args: args{
				rrr: &mswkn.RedditReplyRequest{
					Name: "foo",
					WKNs: []string{"AABBCC", "CCCCCC", "DDDDDD", "GGGGGG"},
					Securities: map[string]*mswkn.Security{
						"AABBCC": {
							Name:           "a",
							WKN:            "AABBCC",
							Underlying:     "CCCCCC",
							Type:           mswkn.SecurityTypeWarrant,
							WarrantType:    mswkn.SecurityWarrantTypePut,
							WarrantSubType: mswkn.SecurityWarrantSubTypeKnockout,
							Strike:         100,
						},
						"CCCCCC": {
							Name:           "c",
							WKN:            "CCCCCC",
							Type:           mswkn.SecurityTypeCommonStock,
							WarrantType:    mswkn.SecurityWarrantTypeUndefined,
							WarrantSubType: mswkn.SecurityWarrantSubTypeUndefined,
						},
						"DDDDDD": {
							Name:           "d",
							WKN:            "DDDDDD",
							Underlying:     "FFFFFF",
							Type:           mswkn.SecurityTypeWarrant,
							WarrantType:    mswkn.SecurityWarrantTypePut,
							WarrantSubType: mswkn.SecurityWarrantSubTypeKnockout,
							Strike:         100,
						},
						"GGGGGG": {
							Name:           "g",
							WKN:            "GGGGGG",
							Underlying:     "HHHHHH",
							Type:           mswkn.SecurityTypeWarrant,
							WarrantType:    mswkn.SecurityWarrantTypePut,
							WarrantSubType: mswkn.SecurityWarrantSubTypeKnockout,
							Strike:         100,
						},
						"HHHHHH": {
							Name:           "h",
							WKN:            "HHHHHH",
							Type:           mswkn.SecurityTypeCommonStock,
							WarrantType:    mswkn.SecurityWarrantTypeUndefined,
							WarrantSubType: mswkn.SecurityWarrantSubTypeUndefined,
						},
					},
					InfoLinks: map[string]*mswkn.InfoLink{
						"AABBCC": {
							WKN: "AABBCC",
							URL: "AABBCC-URL",
						},
						"CCCCCC": {
							WKN: "CCCCCC",
							URL: "CCCCCC-URL",
						},
						"DDDDDD": {
							WKN: "DDDDDD",
							URL: "DDDDDD-URL",
						},
						"GGGGGG": {
							WKN: "GGGGGG",
							URL: "GGGGGG-URL",
						},
					},
				},
			},
			want: []*ReplyLine{
				{
					SecURL:     "[AABBCC](AABBCC-URL)",
					Name:       "a",
					Type:       "KO Put",
					Strike:     "100,00",
					Expire:     "",
					Underlying: "[c CCCCCC](CCCCCC-URL)",
				},
				{
					SecURL:     "[CCCCCC](CCCCCC-URL)",
					Name:       "c",
					Type:       "Stock",
					Strike:     "",
					Expire:     "",
					Underlying: "",
				},
				{
					SecURL:     "[DDDDDD](DDDDDD-URL)",
					Name:       "d",
					Type:       "KO Put",
					Strike:     "100,00",
					Expire:     "",
					Underlying: "FFFFFF",
				},
				{
					SecURL:     "[GGGGGG](GGGGGG-URL)",
					Name:       "g",
					Type:       "KO Put",
					Strike:     "100,00",
					Expire:     "",
					Underlying: "h HHHHHH",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getReplyLines(tt.args.rrr)
			assert.Equal(t, tt.want, got)
		})
	}
}
