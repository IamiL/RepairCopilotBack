package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// InspectionResetter интерфейс для сброса счётчиков проверок
type InspectionResetter interface {
	ResetDailyInspectionsForAllUsers(ctx context.Context) error
}

// Scheduler управляет крон-джобами
type Scheduler struct {
	cron   *cron.Cron
	log    *slog.Logger
	resetter InspectionResetter
}

// New создаёт новый планировщик
func New(log *slog.Logger, resetter InspectionResetter) *Scheduler {
	// Создаём cron с московским часовым поясом
	moscowLocation, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Error("failed to load Moscow timezone, using UTC", "error", err)
		moscowLocation = time.UTC
	}

	c := cron.New(cron.WithLocation(moscowLocation))

	return &Scheduler{
		cron:   c,
		log:    log,
		resetter: resetter,
	}
}

// Start запускает планировщик
func (s *Scheduler) Start() error {
	// Добавляем задачу на полночь по МСК (0 0 * * *)
	_, err := s.cron.AddFunc("0 0 * * *", func() {
		s.log.Info("Starting daily inspections reset job")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.resetter.ResetDailyInspectionsForAllUsers(ctx); err != nil {
			s.log.Error("failed to reset daily inspections", "error", err)
			return
		}

		s.log.Info("Daily inspections reset completed successfully")
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	s.log.Info("Cron scheduler started", "timezone", "Europe/Moscow")

	return nil
}

// Stop останавливает планировщик
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.log.Info("Cron scheduler stopped")
}