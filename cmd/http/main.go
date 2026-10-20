package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/faruqrahmadani/slack-thread-alert/internal/config"
	httpdelivery "github.com/faruqrahmadani/slack-thread-alert/internal/delivery/http"
	"github.com/faruqrahmadani/slack-thread-alert/internal/repository/memory"
	slackrepo "github.com/faruqrahmadani/slack-thread-alert/internal/repository/slack"
	alertuc "github.com/faruqrahmadani/slack-thread-alert/internal/usecase/alert"
	"github.com/faruqrahmadani/slack-thread-alert/pkg/env"
)

func main() {
	// Init ENV, empty falls back to development
	if err := env.Init(os.Getenv("ENV")); err != nil {
		log.Fatalf("error initializing env: %v", err)
	}

	// Load configuration of the current environment
	cfg, err := config.LoadAll(env.GetEnv())
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	log.Printf("configuration loaded: app=%s env=%s port=%s log_level=%s slack_bot_token=%s",
		cfg.App.Name,
		env.GetEnv(),
		cfg.App.HTTPServer.Port,
		cfg.App.Logger.Level,
		mask(cfg.Client.Slack.BotToken),
	)

	// Initialize repositories
	threadRepo := memory.NewThreadRepository(memory.DefaultThreadTTL, memory.DefaultEventTTL)
	slackRepo := slackrepo.NewRepository(cfg.Client.Slack)

	// Initialize usecases
	alertUC := alertuc.NewUseCase(threadRepo, slackRepo)

	// Create and start server
	server := httpdelivery.NewHttpServer(cfg, alertUC)
	go server.ListenAndServe()

	// Handle graceful shutdown
	handleGracefulShutdown(cfg, server)
}

func handleGracefulShutdown(cfg *config.Schema, server httpdelivery.ServerItf) {
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.App.HTTPServer.GracefulTimeout)
	defer cancel()

	server.Stop(ctx)
}

// mask hides a secret, keeping only the last 4 characters
func mask(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}
