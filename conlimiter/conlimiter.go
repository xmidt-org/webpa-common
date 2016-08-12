// Package conlimiter provides a simple connection limiter for incoming TCP
// connections
package conlimiter

import (
	"net"
	"net/http"
	"sync/atomic"
)

// ConLimiter limits the number of outstanding TCP connections coming into
// a HTTP server _before_ TLS is attempted.  This is a defense against a DOS
// attack (intentional or otherwise).
type ConLimiter struct {
	Max     int32
	current int32
}

// Limit is the factor for the ConLimiter package
func (l *ConLimiter) Limit(s *http.Server) {
	s.ConnState = func(c net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			atomic.AddInt32(&l.current, 1)
			if l.Max < atomic.LoadInt32(&l.current) {
				c.Close()
			}
		case http.StateHijacked:
			atomic.AddInt32(&l.current, -1)
		case http.StateClosed:
			atomic.AddInt32(&l.current, -1)
		}
	}
}
