package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

func main() {
	server := New(":13306")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Listen(); err != nil {
			logrus.WithError(err).Fatal("launching server")
		}
	}()

	terminationCh := make(chan os.Signal, 1)

	logrus.Info("started application")
	signal.Notify(terminationCh, syscall.SIGINT, syscall.SIGTERM)

	<-terminationCh
	logrus.Info("closing application")

	if err := server.Close(); err != nil {
		logrus.WithError(err).Error("closing server")
	}

	wg.Wait()

	logrus.Info("closed application")
}

type Server struct {
	http *http.Server
}

func New(serverAddr string) *Server {
	s := &Server{
		http: &http.Server{
			Addr: serverAddr,
		},
	}

	s.http.Handler = s.router()

	return s
}

func (s *Server) Close() error {
	return s.http.Close()
}

func (s *Server) router() chi.Router {
	r := chi.NewRouter()

	r.Get("/events", s.events)
	r.Get("/version", s.version)

	return r
}

func (s *Server) Listen() error {
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Error("serving server")
	}

	return nil
}
