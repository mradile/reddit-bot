package data

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/friendsofgo/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const xetraInstrumentField = 2
const xetraISINField = 3
const xetraWKNField = 6
const xetraInstrumentTypeField = 18
const xetraWarrantSubTypeField = 106
const xetraUnderLyingField = 109
const xetraExpireDateField = 110
const xetraStrikePriceField = 121
const xetraWarrantTypeField = 132

//ToDo add strike column

var httpClient = http.Client{
	Timeout: time.Second * 60,
}

/*
https://www.xetra.com/resource/blob/1643458/1a32c7f3fd2a88ce39bfed00e2a02ca6/data/t7-xfra-allTradableInstruments.zip
*/

var xetraCSVRegex = regexp.MustCompile(`href="(\/resource\/blob\/\w+\/\w+\/data\/t7-xfra-allTradableInstruments\.zip)"`)

const xetraHTMLURL = "https://www.xetra.com/xetra-de/instrumente/alle-handelbaren-instrumente/boersefrankfurt"
const xetraBaseURL = "https://www.xetra.com/"

func LoadXetraCSV(ctx context.Context, secChan chan *mswkn.Security) error {
	dlURL, err := getXetraCSVDownloadURL(ctx)
	if err != nil {
		return fmt.Errorf("could not determine csv download url: %w", err)
	}

	csvBytes, err := downloadXetraCSV(ctx, dlURL)
	if err != nil {
		return fmt.Errorf("could not download csv file: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(csvBytes), int64(len(csvBytes)))
	if err != nil {
		return fmt.Errorf("could not create zip reader: %w", err)
	}

	if len(zipReader.File) != 1 {
		return fmt.Errorf("invalid amount of files found in zip: %d", len(zipReader.File))
	}

	f, err := zipReader.File[0].Open()
	if err != nil {
		return fmt.Errorf("could not open zip file")
	}
	defer f.Close()

	return ParseXetraCSV(f, secChan)
}

func downloadXetraCSV(ctx context.Context, csvURL string) ([]byte, error) {
	lg := log.With().Str("comp", "xetra").Logger()
	lg.Debug().Str("url", csvURL).Msg("trying to download csv")

	ctx, cancel := context.WithTimeout(ctx, httpClient.Timeout-time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, csvURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	lg.Debug().Msg("downloaded csv")

	return ioutil.ReadAll(res.Body)
}

func getXetraCSVDownloadURL(ctx context.Context) (string, error) {
	lg := log.With().Str("comp", "xetra").Logger()
	lg.Debug().Msg("trying to to find csv download url")

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xetraHTMLURL, nil)
	if err != nil {
		return "", err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	html, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	csvLink := xetraCSVRegex.FindSubmatch(html)
	if len(csvLink) != 2 {
		return "", fmt.Errorf("invalid match length: %d", len(csvLink))
	}

	if len(csvLink) < 1 {
		return "", errors.New("could not find xetra csv link")
	}

	u := fmt.Sprintf("%s%s", xetraBaseURL, csvLink[1])

	lg.Debug().Str("url", u).Msg("found csv download url")

	return u, nil
}

func ParseXetraCSV(csvFile io.Reader, secChan chan *mswkn.Security) error {
	lg := log.With().Str("comp", "xetra").Logger()

	loader := csv.NewReader(csvFile)
	loader.Comma = ';'
	loader.FieldsPerRecord = 138

	for {
		line, err := loader.Read()
		if e, ok := err.(*csv.ParseError); ok {
			//skip first three rows
			if e.Err == csv.ErrFieldCount {
				continue
			}
		}
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		//csv header line
		if line[0] == "Product Status" {
			continue
		}

		isin := strings.ToUpper(line[xetraISINField])
		fullWKN := strings.ToUpper(line[xetraWKNField])
		wkn := shortenWKN(fullWKN)
		warrantType := getWarrantType(line[xetraWarrantTypeField])
		warrantSubType := getWarrantSubType(line[xetraWarrantSubTypeField])

		sec := &mswkn.Security{
			Name:           line[xetraInstrumentField],
			ISIN:           isin,
			WKN:            wkn,
			Type:           getSecurityType(line[xetraInstrumentTypeField]),
			Underlying:     line[xetraUnderLyingField],
			WarrantType:    warrantType,
			WarrantSubType: warrantSubType,
		}

		if s := line[xetraStrikePriceField]; s != "" {
			strike, err := strconv.ParseFloat(s, 64)
			if err == nil {
				sec.Strike = strike
			}
		}

		if s := line[xetraExpireDateField]; s != "" {
			if t, err := time.Parse("2006-01-02", line[xetraExpireDateField]); err == nil {
				sec.Expire = &t
			}
		}

		if zerolog.GlobalLevel() == zerolog.TraceLevel {
			lg.Trace().Str("wkn", wkn).Msgf("CSV: %s", strings.Join(line, ";"))
			lg.Trace().Str("wkn", wkn).Msgf("SEC: %+v", sec)
		}

		secChan <- sec
	}
	close(secChan)

	return nil
}

func getWarrantSubType(wType string) int {
	switch wType {
	case "60":
		return mswkn.SecurityWarrantSubTypeKnockout
	case "50":
		return mswkn.SecurityWarrantSubTypeKnockout
	case "40":
		return mswkn.SecurityWarrantSubTypeOS
	}
	return mswkn.SecurityWarrantSubTypeUndefined
}

func getWarrantType(wType string) int {
	switch strings.ToUpper(wType) {
	case "CALL":
		return mswkn.SecurityWarrantTypeCall
	case "PUT":
		return mswkn.SecurityWarrantTypePut
	}
	return mswkn.SecurityWarrantTypeUndefined
}

func getSecurityType(xetraStr string) int {
	switch strings.ToUpper(xetraStr) {
	case "OPT":
		return mswkn.SecurityTypeOption
	case "FUT":
		return mswkn.SecurityTypeFuture
	case "BOND":
		return mswkn.SecurityTypeBond
	case "CS":
		return mswkn.SecurityTypeCommonStock
	case "ETF":
		return mswkn.SecurityTypeExchangeTradedFund
	case "ETC":
		return mswkn.SecurityTypeExchangeTradedCommodity
	case "WAR":
		return mswkn.SecurityTypeWarrant
	case "ETN":
		return mswkn.SecurityTypeExchangeTradedNode
	}
	return mswkn.SecurityTypeUndefined
}

func shortenWKN(long string) string {
	short := long[3:]
	return short
}
