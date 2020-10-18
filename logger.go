package middleware

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Logger interface {
	ListeningOn(string)
	Shutdown(error)
	Log(AccessLog)
}

type AccessLog struct {
	StartTime     time.Time
	Host          string
	RemoteAddr    string
	RemoteUser    string
	Method        string
	Path          string
	Query         url.Values
	Protocol      string
	StatusCode    int
	BytesReceived int64
	BytesSent     int
	Header        http.Header
	Duration      time.Duration
}

type EmptyLogger struct{}

func (l *EmptyLogger) ListeningOn(addr string) {}

func (l *EmptyLogger) Shutdown(err error) {}

func (l *EmptyLogger) Log(data AccessLog) {}

type BasicLogger struct {
	logger *log.Logger
}

func NewBasicLogger() Logger {
	return &BasicLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *BasicLogger) ListeningOn(addr string) {
	l.logger.Println("listening on", addr)
}

func (l *BasicLogger) Shutdown(err error) {
	if err != nil {
		l.logger.Fatal("logger.shutdown", err)
		return
	}
	l.logger.Println("server shutdown")
}

func (l *BasicLogger) Log(data AccessLog) {
	fullURL := data.Path
	if params := data.Query.Encode(); params != "" {
		fullURL += "?" + params
	}
	l.logger.Printf(
		"%s %s \"%s %s %s\" %d %d \"%s\" %v",
		data.Host,
		data.RemoteAddr,
		data.Method,
		fullURL,
		data.Protocol,
		data.StatusCode,
		data.BytesSent,
		data.Header.Get("User-Agent"),
		data.Duration,
	)
}
