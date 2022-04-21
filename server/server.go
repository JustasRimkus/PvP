package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Server struct {
	http         *http.Server
	incomingAddr string
	targetAddr   string
	metrics      metrics
}

type metrics struct {
	totalProcessed prometheus.Counter
}

func New(incomingAddr, targetAddr, serverAddr string) *Server {
	s := &Server{
		http: &http.Server{
			Addr: serverAddr,
		},
		incomingAddr: incomingAddr,
		targetAddr:   targetAddr,
		metrics: metrics{
			totalProcessed: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "app",
				Subsystem: "api",
				Name:      "total_processed_requests",
			}),
		},
	}

	s.http.Handler = s.router()
	prometheus.MustRegister(s.metrics.totalProcessed)

	return s
}

func (s *Server) Close() error {
	return s.http.Close()
}

func (s *Server) router() chi.Router {
	r := chi.NewRouter()

	r.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))

	return r
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.incomingAddr)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := s.http.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("unexpected server closure error")
		}

		if err := ln.Close(); err != nil {
			logrus.WithError(err).Fatal("listener closure error")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			s.metrics.totalProcessed.Inc()
		}
	}()

	for ctx.Err() == nil {
		conn, err := ln.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				logrus.WithError(err).Error("accepting a new connection")
			}

			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleConn(ctx, conn)
		}()

		s.metrics.totalProcessed.Inc()
	}

	wg.Wait()

	return nil
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	rconn, err := net.Dial("tcp", s.targetAddr)
	if err != nil {
		return
	}

	cctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		connect(cctx, conn, rconn)
		cancel()
	}()

	go func() {
		defer wg.Done()
		connect(cctx, rconn, conn)
		cancel()
	}()

	wg.Wait()

	rconn.Close()
	conn.Close()
}

func connect(ctx context.Context, src, dst net.Conn) {
	buff := make([]byte, 65535)
	for ctx.Err() == nil {
		n, err := src.Read(buff)
		if err != nil {
			logrus.WithError(err).Error("reading incoming data")
			return
		}
		b := buff[:n]

		logrus.WithField("data", string(b)).Info("received a message")

		_, err = dst.Write(b)
		if err != nil {
			logrus.WithError(err).Error("writing incoming data")
			return
		}
	}
}
