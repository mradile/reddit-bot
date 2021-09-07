package mswkn

import (
	"context"
	"errors"
)

var (
	ErrInfoLinkNotFound = errors.New("info link not found")
)

type InfoLink struct {
	WKN string
	URL string
}

type InfoLinkRepository interface {
	Add(ctx context.Context, il *InfoLink) error
	Get(ctx context.Context, wkn string) (*InfoLink, error)
}
