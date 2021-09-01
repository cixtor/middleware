package middleware

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// startServer setups and starts the web server.
func (m *Middleware) startServer(addr string, f func() error) error {
	if err := m.validateHostAndPort(addr); err != nil {
		return err
	}

	m.serverShutdown = make(chan bool)
	m.serverInstance = &http.Server{
		Addr:              addr,
		Handler:           m,
		ReadTimeout:       m.ReadTimeout,
		ReadHeaderTimeout: m.ReadHeaderTimeout,
		WriteTimeout:      m.WriteTimeout,
		IdleTimeout:       m.IdleTimeout,
		ErrorLog:          m.ErrorLog,
	}

	go m.gracefulServerShutdown()

	m.Logger.ListeningOn(addr)

	return f() /* ListenAndServe OR ListenAndServeTLS */
}

var ErrInvalidAddressFormat = errors.New("server address must be [string]:[0-65535]")

var ErrHostnameIsTooLong = errors.New("a valid hostname has a maximum of 253 ASCII characters")

var ErrInvalidHostnameLabel = errors.New("each hostname label must be between 1-63 characters long")

var ErrInvalidPortSyntax = errors.New("cannot parse port number due to invalid syntax")

var ErrInvalidPortNumber = errors.New("port number must be in the range [0:65535]")

// validateHostAndPort returns an error if either the hostname or port number
// in the server address is invalid.
//
// Hostname (archaically nodename) is a label that is assigned to a device
// connected to a computer network and that is used to identify the device in
// various forms of electronic communication, such as the World Wide Web.
// Hostnames may be simple names consisting of a single word or phrase, or they
// may be structured. Each hostname usually has at least one numeric network
// address associated with it for routing packets for performance and other
// reasons.
//
// Internet hostnames may have appended the name of a Domain Name System (DNS)
// domain, separated from the host-specific label by a period ("dot"). In the
// latter form, a hostname is also called a domain name. If the domain name is
// completely specified, including a top-level domain of the Internet, then the
// hostname is said to be a fully qualified domain name (FQDN). Hostnames that
// include DNS domains are often stored in the Domain Name System together with
// the IP addresses of the host they represent for the purpose of mapping the
// hostname to an address, or the reverse process.
//
// Hostnames are composed of a sequence of labels concatenated with dots. For
// example, "en.example.org" is a hostname. Each label must be from 1 to 63
// characters long. The entire hostname, including the delimiting dots, has a
// maximum of 253 ASCII characters.
//
// Reference: https://en.wikipedia.org/wiki/Hostname
//
// Port is a communication endpoint. At the software level, within an operating
// system, a port is a logical construct that identifies a specific process or
// a type of network service. A port is identified for each transport protocol
// and address combination by a 16-bit unsigned number, known as the port
// number.
//
// A port number is a 16-bit unsigned integer, thus ranging from 0 to 65535.
//
// For TCP, port number 0 is reserved and cannot be used, while for UDP, the
// source port is optional and a value of zero means no port.
//
// A port number is always associated with an IP address of a host and the type
// of transport protocol used for communication. It completes the destination
// or origination network address of a message. Specific port numbers are
// reserved to identify specific services so that an arriving packet can be
// easily forwarded to a running application. For this purpose, port numbers
// lower than 1024 identify the historically most commonly used services and
// are called the well-known port numbers. Higher-numbered ports are available
// for general use by applications and are known as ephemeral ports.
//
// Reference: https://en.wikipedia.org/wiki/Port_%28computer_networking%29
func (m *Middleware) validateHostAndPort(addr string) error {
	parts := strings.Split(addr, ":")

	if len(parts) != 2 {
		return ErrInvalidAddressFormat
	}

	if err := m.validatePort(parts[1]); err != nil {
		return err
	}

	return m.validateHost(parts[0])
}

func (m *Middleware) validateHost(host string) error {
	// ignore cases like ":8080"
	if host == "" {
		return nil
	}

	if len(host) > 253 {
		return ErrHostnameIsTooLong
	}

	p := strings.Split(host, ".")

	for i := 0; i < len(p); i++ {
		if p[i] == "" || len(p[i]) > 63 {
			return ErrInvalidHostnameLabel
		}
	}

	return nil
}

func (m *Middleware) validatePort(port string) error {
	num, err := strconv.Atoi(port)

	if err != nil {
		return ErrInvalidPortSyntax
	}

	if num < 0 || num > 65535 {
		return ErrInvalidPortNumber
	}

	return nil
}

// ListenAndServe listens on a TCP network address and then calls server.Serve
// to handle requests on incoming connections. All accepted connections are
// configured to enable TCP keep-alives. If the hostname is blank, ":http" is
// used. The method always returns a non-nil error.
func (m *Middleware) ListenAndServe(addr string) error {
	return m.startServer(addr, func() error {
		return m.serverInstance.ListenAndServe()
	})
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (m *Middleware) ListenAndServeTLS(addr string, certFile string, keyFile string, cfg *tls.Config) error {
	return m.startServer(addr, func() error {
		m.serverInstance.TLSConfig = cfg /* TLS configuration */
		return m.serverInstance.ListenAndServeTLS(certFile, keyFile)
	})
}

// Shutdown gracefully shuts down the server without interrupting any active
// connections. Shutdown works by first closing all open listeners, then
// closing all idle connections, and then waiting indefinitely for connections
// to return to idle and then shut down.
//
// If the provided context expires before the shutdown is complete, Shutdown
// returns the context's error, otherwise it returns any error returned from
// closing the Server's underlying Listener(s).
func (m *Middleware) Shutdown() {
	m.serverShutdown <- true
}

// gracefulServerShutdown shutdowns the server.
func (m *Middleware) gracefulServerShutdown() {
	<-m.serverShutdown /* wait shutdown */

	ctx, cancel := context.WithTimeout(context.Background(), m.ShutdownTimeout)

	defer cancel()

	m.Logger.Shutdown(m.serverInstance.Shutdown(ctx))
}
