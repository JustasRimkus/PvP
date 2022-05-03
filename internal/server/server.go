package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/JustasRimkus/PvP/internal/infobip"
	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var Debug = false

type Server struct {
	http         *http.Server
	messenger    *infobip.Messenger
	incomingAddr string
	targetAddr   string
	metrics      *metrics
}

type metrics struct {
	totalActiveConnections prometheus.Gauge
	totalReceivedMessages  prometheus.Counter
	totalSentMessages      prometheus.Counter
}

func New(
	incomingAddr, targetAddr, serverAddr string,
	messenger *infobip.Messenger,
) *Server {

	s := &Server{
		http: &http.Server{
			Addr: serverAddr,
		},
		messenger:    messenger,
		incomingAddr: incomingAddr,
		targetAddr:   targetAddr,
		metrics: &metrics{
			totalActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "app",
				Subsystem: "api",
				Name:      "total_active_connections",
			}),
			totalReceivedMessages: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "app",
				Subsystem: "api",
				Name:      "total_received_messages",
			}),
			totalSentMessages: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "app",
				Subsystem: "api",
				Name:      "total_sent_messages",
			}),
		},
	}

	s.http.Handler = s.router()
	prometheus.MustRegister(s.metrics.totalActiveConnections)
	prometheus.MustRegister(s.metrics.totalReceivedMessages)
	prometheus.MustRegister(s.metrics.totalSentMessages)

	return s
}

func (s *Server) Close() error {
	return s.http.Close()
}

func (s *Server) router() chi.Router {
	r := chi.NewRouter()

	r.Handle("/metrics", promhttp.Handler())

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
			s.metrics.totalActiveConnections.Inc()
			s.handleConn(ctx, conn)
			s.metrics.totalActiveConnections.Dec()
		}()
	}

	wg.Wait()

	return nil
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	rconn, err := net.Dial("tcp", s.targetAddr)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.connect(ctx, conn, rconn, true)
		rconn.Close()
		conn.Close()
	}()

	go func() {
		defer wg.Done()
		s.connect(ctx, rconn, conn, false)
		rconn.Close()
		conn.Close()
	}()

	wg.Wait()
}

func (s *Server) connect(ctx context.Context, src, dst net.Conn, receiver bool) {
	buff := make([]byte, 65535)
	for ctx.Err() == nil {
		n, err := src.Read(buff)
		if err != nil {
			if !errors.Is(err, io.EOF) || !errors.Is(err, net.ErrClosed) {
				logrus.WithError(err).Error("reading incoming data")
			}

			return
		}

		b := buff[:n]

		if Debug {
			logrus.WithField("data", string(b)).Info("received a message")
		}

		_, err = dst.Write(b)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				logrus.WithError(err).Error("writing incoming data")
			}

			return
		}

		if receiver {
			s.metrics.totalReceivedMessages.Inc()
		} else {
			s.metrics.totalSentMessages.Inc()
		}
	}
}
