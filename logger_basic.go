package middleware

import (
	"log"
	"net"
	"os"
)

// BasicLogger implements the Logger interface and the NCSA_HTTPd log format.
type BasicLogger struct {
	logger *log.Logger
}

// NewBasicLogger returns a new instance of a basic server access logger.
func NewBasicLogger() Logger {
	return &BasicLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ListeningOn implements the ListeningOn method for the Logger interface.
func (l BasicLogger) ListeningOn(addr net.Addr) {
	l.logger.Println("listening on", addr)
}

// Shutdown implements the Shutdown method for the Logger interface.
func (l BasicLogger) Shutdown(err error) {
	if err != nil {
		l.logger.Fatalf("server closed (err=%s)", err)
		return
	}

	l.logger.Println("server closed", "(ok)")
}

// Log implements the Log method for the Logger interface.
func (l BasicLogger) Log(data AccessLog) {
	l.logger.Println(data)
}
