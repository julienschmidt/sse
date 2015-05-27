// Copyright 2015 Julien Schmidt. All rights reserved.
// Use of this source code is governed by MIT license, a copy can be found
// in the LICENSE file.

// Package sse provides HTML5 Server-Sent Events for Go
package sse

import (
	"fmt"
	"net/http"
)

type client chan string

// Streamer receives Events and broadcasts them to all connected clients.
//
// Events can be send via the Event channel:
//  streamer.Event <- "Test"
type Streamer struct {
	Event         chan string
	clients       map[client]bool
	connecting    chan client
	disconnecting chan client
}

// New returns a new initialized SSE Streamer
func New() *Streamer {
	s := &Streamer{
		Event:         make(chan string, 1),
		clients:       make(map[client]bool),
		connecting:    make(chan client),
		disconnecting: make(chan client),
	}

	s.run()
	return s
}

// run starts a goroutine to handle client connects and broadcast events.
func (s *Streamer) run() {
	go func() {
		for {
			select {
			case cl := <-s.connecting:
				s.clients[cl] = true

			case cl := <-s.disconnecting:
				delete(s.clients, cl)

			case event := <-s.Event:
				for cl := range s.clients {
					cl <- event
				}
			}
		}
	}()
}

// ServeHTTP implements http.Handler interface.
func (s *Streamer) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	// We need to be able to flush for SSE
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Flushing not supported", http.StatusInternalServerError)
		return
	}

	// Returns a channel that blocks until the connection is closed
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "Closing not supported", http.StatusInternalServerError)
		return
	}
	close := cn.CloseNotify()

	// Set headers for SSE
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	// Connect new client
	cl := make(client)
	s.connecting <- cl

	for {
		select {
		case <-close:
			// Disconnect the client when the connection is closed
			s.disconnecting <- cl
			return

		case event := <-cl:
			// Write events
			fmt.Fprintf(w, "data: %s\n\n", event)
			fl.Flush()
		}
	}
}
