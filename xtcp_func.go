package xtcp

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"time"
)

const (
	defaultConnTimeout    = 30 * time.Second
	defaultRetryInternal  = 100 * time.Millisecond
	defaultReadBufferSize = 128
)

type Retry struct {
	Count    int
	Interval time.Duration
}

func NewNetConn(addr string, timeout ...time.Duration) (net.Conn, error) {
	d := defaultConnTimeout
	if len(timeout) > 0 {
		d = timeout[0]
	}
	return net.DialTimeout("tcp", addr, d)
}

func NewNetConnTLS(addr string, tlsConfig *tls.Config, timeout ...time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: defaultConnTimeout,
	}
	if len(timeout) > 0 {
		dialer.Timeout = timeout[0]
	}
	return tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
}

func NewNetConnKeyCrt(addr, crtFile, keyFile string, timeout ...time.Duration) (net.Conn, error) {
	tlsConfig, err := LoadKeyCrt(crtFile, keyFile)
	if err != nil {
		return nil, err
	}
	return NewNetConnTLS(addr, tlsConfig, timeout...)
}

func LoadKeyCrt(crtPath, keyPath string) (*tls.Config, error) {
	crt, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{crt}
	tlsConfig.Time = time.Now
	tlsConfig.Rand = rand.Reader
	return tlsConfig, nil
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if netError, ok := err.(net.Error); ok && netError.Timeout() {
		return true
	}
	return false
}
