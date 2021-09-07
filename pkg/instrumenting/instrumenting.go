package instrumenting

import (
	"sync"
	"time"
)

type Instrumenting struct {
	lock             sync.Mutex
	dataUpdateStatus *DataUpdateStatus
}

type DataUpdateStatus struct {
	LastUpdated time.Time `json:"last_updated"`
}

func (s *DataUpdateStatus) Status() bool {
	dur := time.Since(s.LastUpdated).Nanoseconds()
	return dur <= int64(time.Hour*2)
}

var Status = &Instrumenting{
	lock:             sync.Mutex{},
	dataUpdateStatus: &DataUpdateStatus{},
}

func (i *Instrumenting) SetDataUpdateStatus(s *DataUpdateStatus) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.dataUpdateStatus = s
}

func (i *Instrumenting) GetDataUpdateStatus() *DataUpdateStatus {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.dataUpdateStatus
}
