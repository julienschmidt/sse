// Copyright 2015 Julien Schmidt. All rights reserved.
// Use of this source code is governed by MIT license,
// a copy can be found in the LICENSE file.

package sse

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"testing"
	"time"
)

type mockResponseWriter struct {
	header  http.Header
	written string
	status  int
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

func NewMockRequest() (*http.Request, context.CancelFunc) {
	request, err := http.NewRequest("GET", "MOCK", nil)
	if err != nil {
		panic(err)
	}
	context, cancel := context.WithCancel(context.Background())
	return request.WithContext(context), cancel
}

func NewMockRequestNeverClose() *http.Request {
	request, err := http.NewRequest("GET", "MOCK", nil)
	if err != nil {
		panic(err)
	}
	return request
}

func NewMockRequestWithTimeout(d time.Duration) *http.Request {
	request, err := http.NewRequest("GET", "MOCK", nil)
	if err != nil {
		panic(err)
	}
	context, _ := context.WithTimeout(context.Background(), d)
	return request.WithContext(context)
}

// all those good old Java times...
type mockResponseWriteFlushCloser struct {
	mockResponseWriteFlusher
}

func NewMockResponseWriteFlushCloser() *mockResponseWriteFlushCloser {
	return &mockResponseWriteFlushCloser{
		NewMockResponseWriteFlusher(),
	}
}

func TestNoFlush(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriter()

	time.Sleep(500 * time.Millisecond)

	streamer.ServeHTTP(w, NewMockRequestWithTimeout(500*time.Millisecond))

	if w.status != http.StatusNotImplemented {
		t.Fatal("wrong status code:", w.status)
	}
	if w.written != "Flushing not supported\n" {
		t.Fatal("wrong error, got:", w.written)
	}
}

func TestClose(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriteFlusher()

	time.Sleep(500 * time.Millisecond)

	streamer.ServeHTTP(w, NewMockRequestWithTimeout(time.Millisecond))

	if w.status != http.StatusOK {
		t.Fatal("wrong status code:", w.status)
	}
	if w.written != "" {
		t.Fatal("wrong error, got:", w.written)
	}
}

func TestClientConnection(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriteFlushCloser()
	r, cancel := NewMockRequest()

	time.Sleep(500 * time.Millisecond)
	go func() {
		time.Sleep(500 * time.Millisecond)
		if len(streamer.clients) != 1 {
			t.Fatal("expected 1 client, has:", len(streamer.clients))
		}
		cancel()
	}()

	if len(streamer.clients) != 0 {
		t.Fatal("expected 0 clients, has:", len(streamer.clients))
	}
	streamer.ServeHTTP(w, r)

	time.Sleep(500 * time.Millisecond)
	if len(streamer.clients) != 0 {
		t.Fatal("expected 0 clients, has:", len(streamer.clients))
	}

	if w.status != http.StatusOK {
		t.Fatal("wrong status code:", w.status)
	}
}

func TestHeader(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriteFlushCloser()
	r, cancel := NewMockRequest()

	time.Sleep(500 * time.Millisecond)
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	streamer.ServeHTTP(w, r)

	if w.status != http.StatusOK {
		t.Fatal("wrong status code:", w.status)
	}

	var expected = []struct {
		header string
		value  string
	}{
		{"Cache-Control", "no-cache"},
		{"Connection", "keep-alive"},
		{"Content-Type", "text/event-stream"},
	}
	h := w.Header()

	for _, header := range expected {
		if h.Get(header.header) != header.value {
			t.Errorf(
				"wrong header value for '%s', expected: '%s', got: '%s'",
				header.header,
				header.value,
				h.Get(header.header),
			)
		}
	}
}

func TestSendEvent(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriteFlushCloser()
	r, cancel := NewMockRequest()

	var expected string

	time.Sleep(500 * time.Millisecond)
	go func() {
		time.Sleep(500 * time.Millisecond)

		streamer.SendString("", "", "")
		expected += "data\n\n"

		streamer.SendString("", "", "Test")
		expected += "data:Test\n\n"

		streamer.SendString("", "msg", "Hi!")
		expected += "event:msg\ndata:Hi!\n\n"

		streamer.SendString("", "string", "multi\nline\n\nyay")
		expected += "event:string\ndata:multi\ndata:line\ndata:\ndata:yay\n\n"

		streamer.SendBytes("", "empty", nil)
		expected += "event:empty\ndata\n\n"

		streamer.SendBytes("", "error", []byte("gnah"))
		expected += "event:error\ndata:gnah\n\n"

		streamer.SendBytes("", "", []byte("\nline\nbreak\n\n"))
		expected += "data:\ndata:line\ndata:break\ndata:\ndata:\n\n"

		streamer.SendInt("", "number", math.MaxInt64)
		expected += "event:number\ndata:" + strconv.FormatInt(math.MaxInt64, 10) + "\n\n"

		streamer.SendUint("", "number", math.MaxUint64)
		expected += "event:number\ndata:" + strconv.FormatUint(math.MaxUint64, 10) + "\n\n"

		streamer.SendJSON("", "json", nil)
		expected += "event:json\ndata:null\n\n"

		streamer.SendJSON("", "json", map[string]string{"test": "successful"})
		expected += "event:json\ndata:{\"test\":\"successful\"}\n\n"

		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	streamer.ServeHTTP(w, r)

	if w.status != http.StatusOK {
		t.Fatal("wrong status code:", w.status)
	}

	if w.written != expected {
		t.Fatal("wrong body, got:\n", w.written, "\nexpected:\n", expected)
	}
}

func TestJSONErr(t *testing.T) {
	streamer := New()
	w := NewMockResponseWriteFlushCloser()
	r, cancel := NewMockRequest()

	var expected string
	var err error

	time.Sleep(500 * time.Millisecond)
	go func() {
		time.Sleep(500 * time.Millisecond)

		// Inf can not be marshalled
		err = streamer.SendJSON("", "json", math.Inf(0))

		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	streamer.ServeHTTP(w, r)

	if err == nil {
		t.Fatal("expected an error!")
	}

	if w.status != http.StatusOK {
		t.Fatal("wrong status code:", w.status)
	}

	if w.written != expected {
		t.Fatal("wrong body, got:\n", w.written, "\nexpected:\n", expected)
	}
}
