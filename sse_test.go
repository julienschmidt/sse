// Copyright 2015 Julien Schmidt. All rights reserved.
// Use of this source code is governed by MIT license, a copy can be found
// in the LICENSE file.

package sse

import (
	"net/http"
	"testing"
	"time"
)

type mockResponseWriter struct {
	header      http.Header
	written     string
	status      int
	closeNotify chan bool
}

func (m *mockResponseWriter) Header() (h http.Header) {
	return m.header
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	m.written += string(p)
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	m.written += string(s)
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(code int) {
	m.status = code
}

func (m *mockResponseWriter) Close() {
	if m.closeNotify != nil {
		m.closeNotify <- true
	}
}

func (m *mockResponseWriter) CloseNotify() <-chan bool {
	m.closeNotify = make(chan bool, 1)
	return m.closeNotify
}

func NewMockResponseWriter() *mockResponseWriter {
	m := new(mockResponseWriter)
	m.status = 200
	m.header = http.Header{}
	return m
}

type mockResponseWriteFlusher struct {
	*mockResponseWriter
}

func (m mockResponseWriteFlusher) Flush() {}

func NewMockResponseWriteFlusher() mockResponseWriteFlusher {
	return mockResponseWriteFlusher{NewMockResponseWriter()}
}

func TestNoFlush(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriter()

	time.Sleep(500 * time.Millisecond)

	streamer.ServeHTTP(w, nil)

	if w.status != 500 {
		t.Fatal("expected status code 500, got:", w.status)
	}
}

func TestClientConnection(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriteFlusher()

	time.Sleep(500 * time.Millisecond)
	go func() {
		time.Sleep(500 * time.Millisecond)
		w.Close()
	}()

	streamer.ServeHTTP(w, nil)

	if w.status != 200 {
		t.Fatal("wrong status code:", w.status)
	}
}
