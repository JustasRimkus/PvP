package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/JustasRimkus/PvP/internal/balancer"
	"github.com/JustasRimkus/PvP/internal/infobip"
	"github.com/JustasRimkus/PvP/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

var conf struct {
	Debug        bool          `envconfig:"default=true"`
	Listen       string        `envconfig:"default=:13305"`
	Targets      []string      `envconfig:"default=:13306"`
	Server       string        `envconfig:"default=:13307"`
	BalancerType balancer.Type `envconfig:"default=round-robin"`
	Infobip      struct {
		API              string
		Host             string `envconfig:"default=r541zm.api.infobip.com`
		Recipient        string
		Cooldown         time.Duration `envconfig:"default=1m"`
		ReceiveThreshold int           `envconfig:"default=45"`
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

	bal, err := balancer.New(conf.BalancerType, conf.Targets)
	if err != nil {
		logrus.WithError(err).Fatal("initializing load balancer")
	}

	srv := server.New(
		conf.Listen,
		conf.Server,
		bal,
		infobip.NewMessenger(
			conf.Infobip.API,
			conf.Infobip.Host,
			conf.Infobip.Recipient,
			"Ioflow",
			conf.Infobip.Cooldown,
		),
		conf.Infobip.ReceiveThreshold,
	)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		srv.Limiter(ctx)
	}()

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
