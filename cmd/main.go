package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/JustasRimkus/PvP/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// listen on port 13305 and write to port 13306
	server := server.New(":13305", ":13306", ":13307")

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(ctx); err != nil {
			logrus.WithError(err).Fatal("launching proxy server")
		}
	}()

	terminationCh := make(chan os.Signal, 1)

	logrus.Info("started application")
	signal.Notify(terminationCh, syscall.SIGINT, syscall.SIGTERM)

	<-terminationCh
	logrus.Info("closing application")

	cancel()
	if err := server.Close(); err != nil {
		logrus.WithError(err).Error("closing proxy server")
	}

	wg.Wait()

	logrus.Info("closed application")
}
