package rest

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/mswkn/bot/pkg/config"
	"gitlab.com/mswkn/bot/pkg/instrumenting"
	"net/http"
	"time"
)

func health(conf config.Config) func(c *gin.Context) {
	return func(c *gin.Context) {

		dataStatus := instrumenting.Status.GetDataUpdateStatus()

		type DataStatus struct {
			Status     bool      `json:"status"`
			LastUpdate time.Time `json:"last_update"`
		}
		type Status struct {
			Version    string
			DataStatus DataStatus `json:"data_status"`
		}

		stat := &Status{
			Version: conf.Version,
			DataStatus: DataStatus{
				Status:     dataStatus.Status(),
				LastUpdate: dataStatus.LastUpdated,
			},
		}

		httpCode := 200
		if !dataStatus.Status() {
			httpCode = http.StatusServiceUnavailable
		}

		c.JSON(httpCode, stat)
	}
}
