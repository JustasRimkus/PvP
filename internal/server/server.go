package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/JustasRimkus/PvP/internal/balancer"
	"github.com/JustasRimkus/PvP/internal/core"
	"github.com/JustasRimkus/PvP/internal/infobip"
	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Debug allows to enable debug mode.
var Debug = false

var (
	// bufferSize specifies limiter channel buffer size.
	bufferSize = 128
)

type Server struct {
	http      *http.Server
	messenger *infobip.Messenger
	balancer  balancer.Balancer

	incomingAddr     string
	receiveThreshold int

	sentCh     chan struct{}
	receivedCh chan struct{}
	metrics    *metrics
}

type metrics struct {
	malwarePackets  prometheus.Counter
	receivedPackets prometheus.Counter
	sentPackets     prometheus.Counter
}

func New(
	incomingAddr, serverAddr string,
	bal balancer.Balancer,
	messenger *infobip.Messenger,
	receiveThreshold int,
) *Server {

	s := &Server{
		http: &http.Server{
			Addr: serverAddr,
		},
		messenger:        messenger,
		balancer:         bal,
		incomingAddr:     incomingAddr,
		receiveThreshold: receiveThreshold,
		sentCh:           make(chan struct{}, bufferSize),
		receivedCh:       make(chan struct{}, bufferSize),
		metrics: &metrics{
			malwarePackets: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: core.MetricsNamespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "malware_packets",
			}),
			receivedPackets: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: core.MetricsNamespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "received_packets",
			}),
			sentPackets: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: core.MetricsNamespace,
				Subsystem: core.MetricsSubsystem,
				Name:      "sent_packets",
			}),
		},
	}

	s.http.Handler = s.router()

	prometheus.MustRegister(s.metrics.malwarePackets)
	prometheus.MustRegister(s.metrics.receivedPackets)
	prometheus.MustRegister(s.metrics.sentPackets)

	return s
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
			s.handleConn(ctx, conn)
		}()
	}

	wg.Wait()

	close(s.sentCh)
	close(s.receivedCh)

	return nil
}

func (s *Server) Limiter(ctx context.Context) {
	defer func() {
		for range s.sentCh {
		}
		for range s.receivedCh {
		}
	}()

	tm := time.NewTimer(time.Second)
	defer tm.Stop()

	var (
		totalSent     int
		totalReceived int
		warnings      = 0
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tm.C:
			if totalSent == 0 && totalReceived == 0 {
				continue
			}

			if Debug {
				logrus.WithFields(logrus.Fields{
					"totalSent":     totalSent,
					"totalReceived": totalReceived,
				}).Info("messages per second")
			}

			if totalReceived > s.receiveThreshold {
				warnings++
			} else {
				warnings = 0
			}

			if warnings == 3 {
				message := fmt.Sprintf("Proxy alert, the service is under a heavy workload. Currently receiving %d p/s and sending %d p/s.", totalReceived, totalSent)

				if err := s.messenger.Send(ctx, message); err != nil {
					logrus.WithError(err).Error("sending a sms message")
				} else {
					warnings = 0
				}
			}

			totalSent = 0
			totalReceived = 0
			tm.Reset(time.Second)
		case <-s.sentCh:
			totalSent++
		case <-s.receivedCh:
			totalReceived++
		}
	}
}

func (s *Server) router() chi.Router {
	r := chi.NewRouter()

	r.Handle("/metrics", promhttp.Handler())

	return r
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	targetAddr, cleanup := s.balancer.Pick()
	defer cleanup()

	rconn, err := net.Dial("tcp", targetAddr)
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
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				logrus.WithError(err).Error("reading incoming data")
			}

			return
		}

		b := buff[:n]

		if strings.Contains(string(b), "malware") {
			s.metrics.malwarePackets.Inc()
		}

		_, err = dst.Write(b)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				logrus.WithError(err).Error("writing incoming data")
			}

			return
		}

		if receiver {
			s.metrics.receivedPackets.Inc()

			select {
			case s.receivedCh <- struct{}{}:
				// OK
			default:
				logrus.Error("slow receiver channel")
			}
		} else {
			s.metrics.sentPackets.Inc()

			select {
			case s.sentCh <- struct{}{}:
				// OK
			default:
				logrus.Error("slow sender channel")
			}
		}
	}
}

func (s *Server) Close() error {
	return s.http.Close()
}
