package main

import (
	"context"
	"github.com/cutcut/smartbrain/bus"
	"github.com/cutcut/smartbrain/http"
	"github.com/cutcut/smartbrain/storage"
	"github.com/cutcut/smartbrain/tracker"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Level = logrus.DebugLevel

	return logger
}

func initContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		for sig := range signals {
			switch sig {
			default:
				cancel()
			}
		}
	}()

	return ctx
}

func main() {
	logger := initLogger()

	logger.Info("Starting...")

	storageConfig := storage.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	busConfig := bus.Config{
		Queue:            "smartbrain",
		Url:              "amqp://guest:guest@localhost:5672/",
		ReconnectTimeout: time.Second * 5,
	}

	storageService := storage.InitStorage(storageConfig, time.Second*60*60)
	busService := bus.InitBus(busConfig)
	trackerService := tracker.InitService(logger.WithField("service", "tracker"), storageService, busService, 2)
	ctx := initContext()

	trackerService.Start(ctx)

	if os.Getenv("USE_HTTP") == "true" {
		httpService := http.InitHttp(trackerService, logrus.WithField("service", "http"), "", 8080)
		logger.Info("Start http...")
		httpService.Start()
	}

	logger.Info("Start...")

	<-ctx.Done()

	_ = busService.Shutdown()
	_ = storageService.Shutdown()

	logger.Info("Done")
}
