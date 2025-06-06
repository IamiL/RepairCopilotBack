package app

import (
	"log/slog"
	httpApp "repairCopilotBot/internal/app/http"
	"time"
)

type App struct {
	HTTPServer *httpApp.App
}

type Config struct {
	TokenTTL             time.Duration `yaml:"token_ttl" env-default:"300h"`
	LaboratoryWorkNumber string        `yaml:"laboratory_work_number"`
}

func New(
	log *slog.Logger,
	httpConfig *httpApp.Config,
) *App {

	httpApp := httpApp.New(
		log,
		httpConfig,
	)

	return &App{
		HTTPServer: httpApp,
	}
}

func (a *App) Run() error {
	// Start HTTP server
	go a.HTTPServer.MustRun()

	return nil
}
