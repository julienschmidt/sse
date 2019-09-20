// Copyright 2015 Julien Schmidt. All rights reserved.
// Use of this source code is governed by MIT license,
// a copy can be found in the LICENSE file.

// Package sse provides HTML5 Server-Sent Events for Go.
//
// See http://www.w3.org/TR/eventsource/ for the technical specification
package sse

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type client chan []byte

// Streamer receives events and broadcasts them to all connected clients.
// Streamer is a http.Handler. Clients making a request to this handler receive
// a stream of Server-Sent Events, which can be handled via JavaScript.
// See the linked technical specification for details.
type Streamer struct {
	event         chan []byte
	clients       map[client]bool
	connecting    chan client
	disconnecting chan client
	bufSize       uint
}

// New returns a new initialized SSE Streamer
func New() *Streamer {
	s := &Streamer{
		event:         make(chan []byte, 1),
		clients:       make(map[client]bool),
		connecting:    make(chan client),
		disconnecting: make(chan client),
		bufSize:       2,
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

			case event := <-s.event:
				for cl := range s.clients {
					// TODO: non-blocking broadcast
					//select {
					//case cl <- event: // Try to send event to client
					//default:
					//	fmt.Println("Channel full. Discarding value")
					//}
					cl <- event
				}
			}
		}
	}()
}

// BufSize sets the event buffer size for new clients.
func (s *Streamer) BufSize(size uint) {
	s.bufSize = size
}

func format(id, event string, dataLen int) (p []byte) {
	// calc length
	l := 6 // data\n\n
	if len(event) > 0 {
		l += 6 + len(event) + 1 // event:{event}\n
	}
	if dataLen > 0 {
		l += 1 + dataLen // :{data}
	}

	// build
	p = make([]byte, l)
	i := 0
	if len(event) > 0 {
		copy(p, "event:")
		i += 6 + copy(p[6:], event)
		p[i] = '\n'
		i++
	}
	i += copy(p[i:], "data")
	if dataLen > 0 {
		p[i] = ':'
		i += 1 + dataLen
	}
	copy(p[i:], "\n\n")

	// TODO: id

	return
}

// SendBytes sends an event with the given byte slice interpreted as a string
// as the data value to all connected clients.
// If the id or event string is empty, no id / event type is send.
func (s *Streamer) SendBytes(id, event string, data []byte) {
	dataLen := len(data)
	lfCount := 0

	// We must sent a "data:{data}\n" for each line
	if dataLen > 0 {
		lfCount = bytes.Count(data, []byte("\n"))
		if lfCount > 0 {
			dataLen += (5 * lfCount) // data:
		}
	}

	p := format(id, event, dataLen)

	// fill in data lines
	start := 0
	ins := len(p) - (2 + dataLen)
	for i := 0; lfCount > 0; i++ {
		if data[i] == '\n' {
			copy(p[ins:], data[start:i])
			ins += i - start
			copy(p[ins:], "\ndata:")
			ins += 6

			start = i + 1
			lfCount--
		}
	}
	copy(p[ins:], data[start:])

	s.event <- p
}

// SendInt sends an event with the given int as the data value to all connected
// clients.
// If the id or event string is empty, no id / event type is send.
func (s *Streamer) SendInt(id, event string, data int64) {
	const maxIntToStrLen = 20 // '-' + 19 digits

	p := format(id, event, maxIntToStrLen)
	p = strconv.AppendInt(p[:len(p)-(maxIntToStrLen+2)], data, 10)

	// Re-add \n\n at the end
	p = p[:len(p)+2]
	p[len(p)-2] = '\n'
	p[len(p)-1] = '\n'

	s.event <- p
}

// SendJSON sends an event with the given data encoded as JSON to all connected
// clients.
// If the id or event string is empty, no id / event type is send.
func (s *Streamer) SendJSON(id, event string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	p := format(id, event, len(data))
	copy(p[len(p)-(2+len(data)):], data) // fill in data
	s.event <- p
	return nil
}

// SendString sends an event with the given data string to all connected
// clients.
// If the id or event string is empty, no id / event type is send.
func (s *Streamer) SendString(id, event, data string) {
	dataLen := len(data)
	lfCount := 0

	// We must sent a "data:{data}\n" for each line
	if dataLen > 0 {
		lfCount = strings.Count(data, "\n")
		if lfCount > 0 {
			dataLen += (5 * lfCount) // data:
		}
	}

	p := format(id, event, dataLen)

	// fill in data lines
	start := 0
	ins := len(p) - (2 + dataLen)
	for i := 0; lfCount > 0; i++ {
		if data[i] == '\n' {
			copy(p[ins:], data[start:i])
			ins += i - start
			copy(p[ins:], "\ndata:")
			ins += 6

			start = i + 1
			lfCount--
		}
	}
	copy(p[ins:], data[start:])

	s.event <- p
}

// SendUint sends an event with the given unsigned int as the data value to all
// connected clients.
// If the id or event string is empty, no id / event type is send.
func (s *Streamer) SendUint(id, event string, data uint64) {
	const maxUintToStrLen = 20

	p := format(id, event, maxUintToStrLen)
	p = strconv.AppendUint(p[:len(p)-(maxUintToStrLen+2)], data, 10)

	// Re-add \n\n at the end
	p = p[:len(p)+2]
	p[len(p)-2] = '\n'
	p[len(p)-1] = '\n'

	s.event <- p
}

// ServeHTTP implements http.Handler interface.
func (s *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// We need to be able to flush for SSE
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Flushing not supported", http.StatusNotImplemented)
		return
	}

	// Returns a channel that blocks until the connection is closed
	close := r.Context().Done()

	// Set headers for SSE
	h := w.Header()
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("Content-Type", "text/event-stream")

	// Connect new client
	cl := make(client, s.bufSize)
	s.connecting <- cl

	for {
		select {
		case <-close:
			// Disconnect the client when the connection is closed
			s.disconnecting <- cl
			return

		case event := <-cl:
			// Write events
			w.Write(event) // TODO: error handling
			fl.Flush()
		}
	}
}
