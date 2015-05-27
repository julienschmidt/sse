// Copyright 2015 Julien Schmidt. All rights reserved.
// Use of this source code is governed by MIT license, a copy can be found
// in the LICENSE file.

// Package sse provides HTTP Server-Side-Events for Go
package sse

import (
	"fmt"
	"net/http"
)

type client chan string

var (
	Event         = make(client, 1)
	clients       = make(map[client]bool)
	connecting    = make(chan client)
	disconnecting = make(chan client)
)

var running bool

func Run() {
	// Only run once
	if running {
		return
	}

	// handle client connects and broadcast to clients
	go func() {
		running = true
		for {
			select {
			case cl := <-connecting:
				clients[cl] = true

			case cl := <-disconnecting:
				delete(clients, cl)

			case ev := <-Event:
				for cl, _ := range clients {
					cl <- ev
				}
			}
		}
	}()
}

func Handle(w http.ResponseWriter, _ *http.Request) {
	// We need to be able to flush for SSE
	fl, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set headers for SSE
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	// Connect new client
	cl := make(client)
	connecting <- cl

	// Returns a channel that blocks until the connection is closed
	notify := w.(http.CloseNotifier).CloseNotify()

	for {
		select {
		case <-notify:
			// Disconnect the client when the connection is closed
			disconnecting <- cl
			return

		case e := <-cl:
			// Write events
			fmt.Fprintf(w, "data: %s\n\n", e)
			fl.Flush()
		}
	}
}
