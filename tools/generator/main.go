package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tjarratt/babble"
)

func main() {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())

	gen := New("http://127.0.0.1:13305", 15, 15)

	wg.Add(1)
	go func() {
		defer wg.Done()
		gen.Run(ctx)
	}()

	terminationCh := make(chan os.Signal, 1)

	logrus.Info("started application")
	signal.Notify(terminationCh, syscall.SIGINT, syscall.SIGTERM)

	<-terminationCh
	logrus.Info("closing application")

	cancel()
	wg.Wait()

	logrus.Info("closed application")
}

type Generator struct {
	http       *http.Client
	randomizer *rand.Rand

	targetAddr   string
	basicWorkers int
	sseWorkers   int
}

func New(
	targetAddr string,
	basicWorkers int,
	sseWorkers int,
) *Generator {

	return &Generator{
		http:         &http.Client{},
		randomizer:   rand.New(rand.NewSource(time.Now().Unix())),
		targetAddr:   targetAddr,
		basicWorkers: basicWorkers,
		sseWorkers:   sseWorkers,
	}
}

func (g *Generator) Run(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Add(g.basicWorkers)
	for i := 0; i < g.basicWorkers; i++ {
		go func() {
			defer wg.Done()
			g.startRequester(ctx, 800, g.basicRequest)
		}()
	}

	wg.Add(g.sseWorkers)
	for i := 0; i < g.sseWorkers; i++ {
		go func() {
			defer wg.Done()
			g.startRequester(ctx, 8000, g.sseRequest)
		}()
	}

	wg.Wait()
}

func (g *Generator) startRequester(ctx context.Context, delay int, fn func(context.Context)) {
	durationFn := func() time.Duration {
		return time.Duration(g.randomizer.Intn(delay)) * time.Millisecond
	}

	tm := time.NewTimer(durationFn())
	defer tm.Stop()

	for {

		select {
		case <-ctx.Done():
			return
		case <-tm.C:
			fn(ctx)
			tm.Reset(durationFn())
		}
	}
}

func (g *Generator) basicRequest(ctx context.Context) {
	babbler := babble.NewBabbler()
	babbler.Count = g.randomizer.Intn(100)
	babbler.Separator = " "

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/version", g.targetAddr),
		bytes.NewBuffer(
			[]byte(babbler.Babble()),
		),
	)
	if err != nil {
		if !errors.Is(err, ctx.Err()) {
			logrus.WithError(err).Error("constructing new request")
		}

		return
	}

	resp, err := g.http.Do(req)
	if err != nil {
		if !errors.Is(err, ctx.Err()) {
			logrus.WithError(err).Error("sending http request")
		}

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.WithField("statusCode", resp.StatusCode).
			Error("invalid events status code")
	}
}

func (g *Generator) sseRequest(ctx context.Context) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/events", g.targetAddr),
		http.NoBody,
	)
	if err != nil {
		if !errors.Is(err, ctx.Err()) {
			logrus.WithError(err).Error("constructing new request")
		}

		return
	}

	resp, err := g.http.Do(req)
	if err != nil {
		if !errors.Is(err, ctx.Err()) {
			logrus.WithError(err).Error("sending http request")
		}

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.WithField("statusCode", resp.StatusCode).
			Error("invalid events status code")

		return
	}

	r := bufio.NewReader(resp.Body)
	for {
		_, err := r.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, ctx.Err()) {
				logrus.WithError(err).Error("reading stream")
			}

			return
		}
	}
}
