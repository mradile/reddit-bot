package db

import (
	"context"
	"gitlab.com/mswkn/bot"
	"strings"
	"sync"
)

type MemorySecurityRepository struct {
	//isinLookup map[string]*mswkn.Security
	wknLookup map[string]*mswkn.Security
	lock      sync.Mutex
}

func NewMemorySecurityRepository() mswkn.SecurityRepository {
	m := &MemorySecurityRepository{
		//	isinLookup: make(map[string]*mswkn.Security),
		wknLookup: make(map[string]*mswkn.Security),
		lock:      sync.Mutex{},
	}
	return m
}

func (m *MemorySecurityRepository) Add(ctx context.Context, sec *mswkn.Security) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	//m.isinLookup[strings.ToUpper(sec.ISIN)] = sec
	m.wknLookup[strings.ToUpper(sec.WKN)] = sec
	return nil
}

func (m *MemorySecurityRepository) AddBulk(ctx context.Context, secs []*mswkn.Security) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, sec := range secs {
		//m.isinLookup[strings.ToUpper(sec.ISIN)] = sec
		m.wknLookup[strings.ToUpper(sec.WKN)] = sec
	}
	return nil
}

func (m *MemorySecurityRepository) Get(ctx context.Context, wkn string) (*mswkn.Security, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	sec, ok := m.wknLookup[strings.ToUpper(wkn)]
	if !ok {
		return nil, mswkn.ErrSecurityNotFound
	}
	return sec, nil
}

type InfoLinkRepository struct {
	list map[string]*mswkn.InfoLink
	lock sync.Mutex
}

func NewMemoryInfoLinkRepository() mswkn.InfoLinkRepository {
	i := &InfoLinkRepository{
		list: make(map[string]*mswkn.InfoLink),
		lock: sync.Mutex{},
	}
	return i
}

func (i *InfoLinkRepository) Add(ctx context.Context, il *mswkn.InfoLink) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.list[strings.ToUpper(il.WKN)] = il
	return nil
}

func (i *InfoLinkRepository) Get(ctx context.Context, wkn string) (*mswkn.InfoLink, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	il, ok := i.list[strings.ToUpper(wkn)]
	if !ok {
		return nil, mswkn.ErrInfoLinkNotFound
	}
	return il, nil
}
