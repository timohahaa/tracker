package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres-драйвер для database/sql
	log "github.com/sirupsen/logrus"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/migrations"
)

// App — собранное приложение: подключение к БД и HTTP-сервер.
type App struct {
	cfg    config.Config
	db     *sqlx.DB
	server *http.Server
}

// New поднимает подключение к БД, применяет миграции и собирает роутер.
func New(cfg config.Config) (*App, error) {
	db, err := sqlx.Connect("postgres", cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := migrations.Up(migrations.Config{
		DSN:      cfg.PostgresDSN,
		Path:     cfg.MigrationsPath,
		Attempts: 5,
		Timeout:  time.Second,
		Logf:     log.Infof,
	}); err != nil {
		return nil, err
	}

	repos := repository.New(db)
	services := service.New(repos, cfg)
	router := handler.New(services)

	return &App{
		cfg: cfg,
		db:  db,
		server: &http.Server{
			Addr:         cfg.HTTPAddr,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
	}, nil
}

// Run запускает HTTP-сервер и блокируется до сигнала завершения.
func (a *App) Run() error {
	log.Infof("HTTP server listening on: %s", a.cfg.HTTPAddr)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Infof("got signal: %s, shutting down", s)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		log.Errorf("shutdown HTTP server: %v", err)
	}

	return a.db.Close()
}
