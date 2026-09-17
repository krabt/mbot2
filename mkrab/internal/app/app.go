package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mkrab/internal/admin"
	"mkrab/internal/broker"
	"mkrab/internal/config"
	"mkrab/internal/data"
)

type Application struct {
	config   *config.Config
	database *data.Database
	broker   *broker.Server
	admin    *admin.Server
}

func New(cfg *config.Config, db *data.Database, server *broker.Server) *Application {
	application := &Application{config: cfg, database: db, broker: server}
	if cfg.Web.IsEnabled() {
		application.admin = admin.New(cfg.Web.Address, db.DB, server)
	}
	return application
}

func (a *Application) Run() error {
	password, created, err := a.database.EnsureInitialRoot()
	if err != nil {
		return err
	}
	if created {
		slog.Warn("initial root user created", "one_time_password", password)
		slog.Warn("change the root password hash in mqtt_user after first login")
	}
	slog.Info("MQTT broker starting")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := a.broker.Run(); err != nil {
		return err
	}
	if a.admin != nil {
		go func() {
			if err := a.admin.Run(); err != nil && err != http.ErrServerClosed {
				slog.Error("admin server stopped", "error", err)
			}
		}()
	}

	<-ctx.Done()
	slog.Info("MQTT broker stopping", "signal", ctx.Err())
	return nil
}

func (a *Application) Close() error {
	if a.admin != nil {
		_ = a.admin.Close()
	}
	return a.broker.Close()
}
