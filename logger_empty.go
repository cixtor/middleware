package middleware

import (
	"net"
)

// emptyLogger implements the Logger interface to discard access logs.
type emptyLogger struct{}

// ListeningOn implements the ListeningOn method for the Logger interface.
func (l emptyLogger) ListeningOn(addr net.Addr) {}

// Shutdown implements the Shutdown method for the Logger interface.
func (l emptyLogger) Shutdown(err error) {}

// Log implements the Log method for the Logger interface.
func (l emptyLogger) Log(data AccessLog) {}

// DiscardLogs writes all the logs to `/dev/null`.
func (m *Middleware) DiscardLogs() {
	m.Logger = &emptyLogger{}
}
