package rest

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/config"
	"net/http"
	"time"
)

type Server struct {
	s            *http.Server
	msg          mswkn.Broker
	securityRepo mswkn.SecurityRepository
	infoLinkRepo mswkn.InfoLinkRepository
}

func NewServer(conf config.Config, msg mswkn.Broker, securityRepo mswkn.SecurityRepository, infoLinkRepo mswkn.InfoLinkRepository) *Server {

	s := &Server{
		msg:          msg,
		securityRepo: securityRepo,
		infoLinkRepo: infoLinkRepo,
	}

	if conf.Mode == "develop" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(gin.Recovery())

	s.routes(router, conf)

	s.s = &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", conf.HTTPServer.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	return s.s.ListenAndServe()

}

func (s *Server) Stop(ctx context.Context) error {
	return s.s.Shutdown(ctx)
}

func (s *Server) routes(router *gin.Engine, conf config.Config) {
	lg := log.With().Str("comp", "rest").Logger()

	router.GET("/health", health(conf))

	var api *gin.RouterGroup

	if conf.HTTPServer.BasicAuthDisabled {
		lg.Info().Msg("basic auth disabled")
		api = router.Group("/api/v1")
	} else {
		if conf.HTTPServer.Username == "" || conf.HTTPServer.Password == "" {
			lg.Fatal().Msg("http server username and password must not be empty")
		}
		lg.Info().Msg("basic auth enabled")
		api = router.Group("/api/v1", gin.BasicAuth(gin.Accounts{
			conf.HTTPServer.Username: conf.HTTPServer.Password,
		}))
	}

	api.POST("/reddit/comment/inject/:name", inject(s.msg, "/inject"))

	api.GET("/security/:wkn", getSecurity(s.securityRepo, "/security"))
	api.GET("/infolink/:wkn", getInfoLink(s.infoLinkRepo, "/infolink"))
}

func getSecurity(repo mswkn.SecurityRepository, route string) func(c *gin.Context) {
	lg := log.With().Str("comp", "rest").Str("route", route).Logger()
	return func(c *gin.Context) {
		wkn := c.Param("wkn")
		sec, err := repo.Get(c.Request.Context(), wkn)
		if err != nil {
			if err == mswkn.ErrSecurityNotFound {
				c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
				return
			}
			lg.Error().Err(err).Str("route", c.Request.URL.String()).Msg("error")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
			return
		}
		c.JSON(http.StatusOK, sec)
	}
}

func getInfoLink(repo mswkn.InfoLinkRepository, route string) func(c *gin.Context) {
	lg := log.With().Str("comp", "rest").Str("route", route).Logger()

	return func(c *gin.Context) {
		wkn := c.Param("wkn")
		sec, err := repo.Get(c.Request.Context(), wkn)
		if err != nil {
			if err == mswkn.ErrInfoLinkNotFound {
				c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
				return
			}
			lg.Error().Err(err).Str("route", c.Request.URL.String()).Msg("error")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
			return
		}
		c.JSON(http.StatusOK, sec)
	}
}

func inject(msg mswkn.Broker, route string) func(c *gin.Context) {
	lg := log.With().Str("comp", "rest").Str("route", route).Logger()

	return func(c *gin.Context) {

		type Body struct {
			Body string `json:"body"`
		}
		var b Body
		if err := c.Bind(&b); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req := mswkn.RedditRequest{
			Name: c.Param("name"),
			Text: b.Body,
		}

		lg.Debug().Msgf("injecting %+v", req)
		if err := msg.Publish(mswkn.BrokerSubjectWKNRequest, req); err != nil {
			lg.Error().Err(err).Str("route", c.Request.URL.String()).Msg("error")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
			return
		}

		c.String(http.StatusNoContent, "")
	}
}
