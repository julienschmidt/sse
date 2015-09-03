# SSE - Server-Sent Events
[![Build Status](https://travis-ci.org/julienschmidt/sse.svg)](https://travis-ci.org/julienschmidt/sse) [![Coverage](http://gocover.io/_badge/github.com/julienschmidt/sse?0)](http://gocover.io/github.com/julienschmidt/sse) [![GoDoc](https://godoc.org/github.com/julienschmidt/sse?status.svg)](https://godoc.org/github.com/julienschmidt/sse)

[HTML5 Server-Sent Events](http://www.w3.org/TR/eventsource/) for Go

## Why you should use Server-Sent-Events
- No need to implement custom protocol (WebSockets), it just uses HTTP
- Convenient JavaScript API, fires easy to handle Events
- Auto-Reconnects
- Unlike WebSockets, only unidirectional (server -> client)

## ToDo
- ID handling
- Improve Client Channel buffering

## Further Readings
- http://www.w3.org/TR/eventsource/
- http://html5doctor.com/server-sent-events/
- http://www.html5rocks.com/en/tutorials/eventsource/basics/
- https://developer.mozilla.org/en-US/docs/Server-sent_events/Using_server-sent_events
