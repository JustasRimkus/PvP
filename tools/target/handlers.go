package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "cannot upgrade response writer", http.StatusBadRequest)
		return
	}

	randomizer := rand.New(rand.NewSource(time.Now().Unix()))

	durationFn := func() time.Duration {
		return time.Duration(randomizer.Intn(500)) * time.Millisecond
	}

	tm := time.NewTimer(durationFn())
	defer tm.Stop()

	for {
		select {
		case tstamp := <-tm.C:
			fmt.Fprintf(w, "%s\n", tstamp)
			tm.Reset(durationFn())
			flusher.Flush()
		case <-ctx.Done():
			fmt.Fprint(w, "disconnected\n")
			flusher.Flush()
			return
		}
	}
}

func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("v1.0.0"))
}
