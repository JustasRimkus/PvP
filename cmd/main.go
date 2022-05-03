package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/JustasRimkus/PvP/internal/infobip"
	"github.com/JustasRimkus/PvP/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

var conf struct {
	Debug   bool   `envconfig:"default=true"`
	Listen  string `envconfig:"default=:13305"`
	Target  string `envconfig:"default=:13306"`
	Server  string `envconfig:"default=:13307"`
	Infobip struct {
		API       string
		Host      string `envconfig:"default=r541zm.api.infobip.com`
		Recipient string
		Cooldown  time.Duration `envconfig:"default=1m"`
	}
}

func main() {
	if err := envconfig.Init(&conf); err != nil {
		logrus.WithError(err).Fatal("initializing environment variables")
	}

	if conf.Debug {
		server.Debug = true
		infobip.Debug = true
	}

	srv := server.New(
		conf.Listen,
		conf.Target,
		conf.Server,
		infobip.NewMessenger(
			conf.Infobip.API,
			conf.Infobip.Host,
			conf.Infobip.Recipient,
			"IoT Proxy",
			conf.Infobip.Cooldown,
		),
	)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(ctx); err != nil {
			logrus.WithError(err).Fatal("running proxy server")
		}
	}()

	terminationCh := make(chan os.Signal, 1)

	logrus.Info("started application")
	signal.Notify(terminationCh, syscall.SIGINT, syscall.SIGTERM)

	<-terminationCh
	logrus.Info("closing application")

	cancel()
	if err := srv.Close(); err != nil {
		logrus.WithError(err).Error("closing proxy server")
	}

	wg.Wait()

	logrus.Info("closed application")
}
