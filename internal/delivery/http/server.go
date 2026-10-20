package http

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/faruqrahmadani/slack-thread-alert/internal/config"
	"github.com/faruqrahmadani/slack-thread-alert/internal/delivery/http/controller"
	"github.com/faruqrahmadani/slack-thread-alert/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ServerItf interface {
	ListenAndServe()
	Stop(ctx context.Context)
}

type Server struct {
	echo *echo.Echo
}

// NewHttpServer creates a new HTTP server instance
func NewHttpServer(cfg *config.Schema, alertUC usecase.AlertUseCaseItf) ServerItf {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.BodyLimit("1M"))

	ctrl := controller.NewController(alertUC)

	registerRouters(e, ctrl)

	setServerObj(e, cfg.App.HTTPServer)

	return &Server{echo: e}
}

// ListenAndServe starts the HTTP server and blocks until it is stopped
func (s *Server) ListenAndServe() {
	log.Printf("starting HTTP server on %s", s.echo.Server.Addr)

	if err := s.echo.StartServer(s.echo.Server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("error starting HTTP server: %v", err)
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) {
	log.Print("shutting down HTTP server")

	if err := s.echo.Shutdown(ctx); err != nil {
		log.Printf("error shutting down HTTP server: %v", err)
		return
	}

	log.Print("HTTP server shutdown complete")
}

// setServerObj configures server-specific parameters
func setServerObj(e *echo.Echo, serverCfg config.HTTPServer) {
	e.Server.Addr = serverCfg.ListenAddress + ":" + serverCfg.Port
	if serverCfg.ReadTimeout > 0 {
		e.Server.ReadTimeout = serverCfg.ReadTimeout
	}
	if serverCfg.WriteTimeout > 0 {
		e.Server.WriteTimeout = serverCfg.WriteTimeout
	}
	if serverCfg.IdleTimeout > 0 {
		e.Server.IdleTimeout = serverCfg.IdleTimeout
	}
}
