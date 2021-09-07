package integration

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/data"
	"gitlab.com/mswkn/bot/pkg/db"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestLoadXetraCSV(t *testing.T) {
	if os.Getenv("XETRA_DL_TEST") != "1" {
		return
	}

	repo := db.NewMemorySecurityRepository()

	ctx := context.Background()
	PrintMemUsage()

	secChan := make(chan *mswkn.Security)
	go func() {
		for sec := range secChan {
			assert.NoError(t, repo.Add(ctx, sec))
		}
	}()

	assert.NoError(t, data.LoadXetraCSV(ctx, secChan))
	PrintMemUsage()
	time.Sleep(time.Second * 2)
	PrintMemUsage()
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
