package scanner

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func Test_scan(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    []string
		wantErr bool
	}{
		{
			name:    "empty",
			text:    "",
			want:    nil,
			wantErr: true,
		},
		{
			name: "single word",
			text: "foo",
			want: nil,
		},
		{
			name: "$wkn at the end",
			text: "foo $wkn",
			want: nil,
		},
		{
			name: "$wkn at the end with whitespace following",
			text: "foo $wkn   ",
			want: nil,
		},
		{
			name: "wkn at beginning",
			text: "$wkn 123456",
			want: []string{"123456"},
		},
		{
			name: "wkn in between",
			text: "foo bar $wkn 123456 bar foo",
			want: []string{"123456"},
		},
		{
			name: "wkn in between upper",
			text: "foo bar $WKN 123456 bar foo",
			want: []string{"123456"},
		},
		{
			name: "multiple wkns in between",
			text: "foo bar $wkn 123456 bar $wkn 55AA77 foo",
			want: []string{"123456", "55AA77"},
		},
		{
			name: "multiple wkns in between lower and upper",
			text: "foo bar $wkn 123456 bar $WKN 55AA77 foo",
			want: []string{"123456", "55AA77"},
		},
		{
			name: "emoji",
			text: `Ich mag diesen Stock 🌿
		$wkn 645932`,
			want: []string{"645932"},
		},
		{
			name: "emoji 2",
			text: `Ich mag diesen Stock 🌿
		$wkn 645932
		`,
			want: []string{"645932"},
		},
		{
			name: "whitspaces 2",
			text: `a  $wkn  645932  b`,
			want: []string{"645932"},
		},
		{
			name: "whitspaces 3",
			text: `a   $wkn   645932   b`,
			want: []string{"645932"},
		},
		{
			name: "whitspaces 4",
			text: `a    $wkn    645932    b`,
			want: []string{"645932"},
		},
		{
			name: "direct $wkn single",
			text: `$645932`,
			want: []string{"645932"},
		},
		{
			name: "direct $wkn two",
			text: `$645932 $TT66UU`,
			want: []string{"645932", "TT66UU"},
		},
		{
			name: "direct $wkn two with spaces",
			text: `  $645932   $TT66UU  `,
			want: []string{"645932", "TT66UU"},
		},
		{
			name: "direct $wkn two with linebreaks",
			text: `
		$645932
		
		$TT66UU`,
			want: []string{"645932", "TT66UU"},
		},
		{
			name: "direct $wkn two with linebreaks and text",
			text: ` lorem ipsum $645932 fo bar
		ipsum lorem
		$TT66UU
		lorem ipsum`,
			want: []string{"645932", "TT66UU"},
		},
		{
			name: "mixed $wkn",
			text: `  $wkn 645932
$645932   asdsads
asdsad $wkn AA77UU

$TT66UU`,
			want: []string{"645932", "TT66UU", "AA77UU"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DollarWKNTokenScan(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("FullWKN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			sort.SliceStable(got, func(i, j int) bool {
				return got[i] < got[j]
			})
			sort.SliceStable(tt.want, func(i, j int) bool {
				return tt.want[i] < tt.want[j]
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWKNDollarFullScan(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    []string
		wantErr bool
	}{
		{
			name: "simple",
			text: `a wkn follows AABB11`,
			want: []string{"AABB11"},
		},
		{
			name: "multiple",
			text: `ZZYY77 multiple terms AA55BB in between 55BBAA`,
			want: []string{"ZZYY77", "AA55BB", "55BBAA"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WKNDollarFullScan(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("WKNDollarFullScan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.SliceStable(got, func(i, j int) bool {
				return got[i] < got[j]
			})
			sort.SliceStable(tt.want, func(i, j int) bool {
				return tt.want[i] < tt.want[j]
			})
			assert.Equal(t, tt.want, got)
		})
	}
}
