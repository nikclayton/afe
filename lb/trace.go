package main

import (
	"context"
	"fmt"
	"net/http/httptrace"
	"time"
)

type httpTraceStats struct {
	// LatencyRequest records the time taken to send the request after the TCP connection is
	// established or reused.
	LatencyRequest time.Duration
	gotConn        time.Time
	wroteRequest   time.Time

	// LatencyResponse records the time taken to receive the response from the backend after
	// the request has been sent.
	LatencyResponse time.Duration
	done            time.Time

	// LatencyTotal records the total time taken to send and receive the response.
	LatencyTotal time.Duration
}

func (s *httpTraceStats) String() string {
	return fmt.Sprintf("Request(%v) Response(%v) Total(%v)", s.LatencyRequest, s.LatencyResponse, s.LatencyTotal)
}

// WithHTTPTrace returns a new context based on the provided context that records httptrace
// statistics in the provided httpTraceStats struct.
func WithHTTPTrace(ctx context.Context, s *httpTraceStats) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(gotConnInfo httptrace.GotConnInfo) {
			s.gotConn = time.Now()
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			s.wroteRequest = time.Now()
			s.LatencyRequest = s.wroteRequest.Sub(s.gotConn)
		},
	})
}

// Done records the time that the request completed.
func (s *httpTraceStats) Done() {
	s.done = time.Now()
	s.LatencyResponse = s.done.Sub(s.wroteRequest)
	s.LatencyTotal = s.done.Sub(s.gotConn)
}
