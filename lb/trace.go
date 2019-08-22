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

	// LatencyBackend records the time taken by the backend to process the request.
	LatencyBackend time.Duration

	// LatencyResponse records the time taken to receive the response from the backend after
	// the request has been sent.
	LatencyResponse      time.Duration
	gotFirstResponseByte time.Time

	// LatencyTotal records the total time taken to send and receive the response.
	LatencyTotal time.Duration
}

func (s *httpTraceStats) String() string {
	return fmt.Sprintf("Request(%v) Backend(%v) Response(%v) Total(%v)",
		s.LatencyRequest, s.LatencyBackend, s.LatencyResponse, s.LatencyTotal)
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
		},

		GotFirstResponseByte: func() {
			s.gotFirstResponseByte = time.Now()
		},
	})
}

// Done records the time that the request completed.
func (s *httpTraceStats) Done() {
	done := time.Now()
	s.LatencyRequest = s.wroteRequest.Sub(s.gotConn)
	s.LatencyResponse = done.Sub(s.gotFirstResponseByte)
	s.LatencyBackend = s.gotFirstResponseByte.Sub(s.wroteRequest)
	s.LatencyTotal = done.Sub(s.gotConn)
}
