package mswkn

import (
	"context"
	"github.com/friendsofgo/errors"
	"time"
)

const (
	SecurityTypeUndefined = iota
	SecurityTypeOption
	SecurityTypeFuture
	SecurityTypeBond
	SecurityTypeCommonStock
	SecurityTypeExchangeTradedFund
	SecurityTypeExchangeTradedCommodity
	SecurityTypeWarrant
	SecurityTypeExchangeTradedNode
)

const (
	SecurityWarrantTypeUndefined = iota
	SecurityWarrantTypeCall
	SecurityWarrantTypePut
)

const (
	SecurityWarrantSubTypeUndefined = iota
	SecurityWarrantSubTypeKnockout
	SecurityWarrantSubTypeOS
)

var (
	ErrSecurityNotFound = errors.New("security not found")
	ErrSecurityRepo     = errors.New("internal database error")
)

type Security struct {
	Name           string
	ISIN           string
	WKN            string
	Underlying     string
	Type           int
	WarrantType    int
	WarrantSubType int
	Strike         float64
	Expire         *time.Time
}

type SecurityRepository interface {
	Add(ctx context.Context, sec *Security) error
	AddBulk(ctx context.Context, secs []*Security) error
	Get(ctx context.Context, wkn string) (*Security, error)
}
