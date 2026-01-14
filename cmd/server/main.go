package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lsy88/uptime-chopper/internal/api"
	"github.com/lsy88/uptime-chopper/internal/config"
	"github.com/lsy88/uptime-chopper/internal/docker"
	"github.com/lsy88/uptime-chopper/internal/monitor"
	"github.com/lsy88/uptime-chopper/internal/notify"
	"github.com/lsy88/uptime-chopper/internal/store"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	st, err := store.NewJSONStore(cfg.DataFilePath)
	if err != nil {
		logger.Fatal("open store", zap.Error(err))
	}

	dockerClient, err := docker.NewClient()
	if err != nil && !errors.Is(err, docker.ErrDockerUnavailable) {
		logger.Fatal("init docker", zap.Error(err))
	}

	notifier := notify.NewDispatcher(cfg.Notifications)

	engine := monitor.NewEngine(monitor.EngineDeps{
		Logger:       logger,
		Store:        st,
		Docker:       dockerClient,
		Notifier:     notifier,
		MaxLogBytes:  cfg.MaxDockerLogBytes,
		DefaultSince: cfg.DefaultDockerLogSince,
	})
	engine.Start()
	defer engine.Stop()

	r := api.NewRouter(api.Deps{
		Logger: logger,
		Store:  st,
		Docker: dockerClient,
		Engine: engine,
		Config: cfg,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("http listening", zap.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
}
