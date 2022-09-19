package xtcp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
)

const (
	defaultServer = "default"
)

type Server struct {
	mu        sync.Mutex
	listen    net.Listener
	address   string
	handler   func(*Conn)
	tlsConfig *tls.Config
}

// 跟据名字映射server
var serverMapping sync.Map

func GetServer(name ...interface{}) *Server {
	serverName := defaultServer
	if len(name) > 0 && name[0] != "" {
		serverName = name[0].(string)
	}
	server := NewServer("", nil)
	v, _ := serverMapping.LoadOrStore(serverName, server)
	return v.(*Server)
}

func NewServer(address string, handler func(*Conn), name ...string) *Server {
	s := &Server{
		address: address,
		handler: handler,
	}
	if len(name) > 0 && name[0] != "" {
		serverMapping.Store(name[0], s)
	}
	return s
}

func NewServerTLS(address string, tlsConfig *tls.Config, handler func(*Conn), name ...string) *Server {
	s := NewServer(address, handler, name...)
	s.SetTLSConfig(tlsConfig)
	return s
}

func NewServerKeyCrt(address, crtFile, keyFile string, handler func(*Conn), name ...string) *Server {
	s := NewServer(address, handler, name...)
	if err := s.SetTLSKeyCrt(crtFile, keyFile); err != nil {
		fmt.Errorf(err.Error())
	}
	return s
}

func (s *Server) SetAddress(address string) {
	s.address = address
}

func (s *Server) SetHandler(handler func(*Conn)) {
	s.handler = handler
}

func (s *Server) SetTLSKeyCrt(crtFile, keyFile string) error {
	tlsConfig, err := LoadKeyCrt(crtFile, keyFile)
	if err != nil {
		return err
	}
	s.tlsConfig = tlsConfig
	return nil
}

func (s *Server) SetTLSConfig(tlsConfig *tls.Config) {
	s.tlsConfig = tlsConfig
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listen == nil {
		return nil
	}
	return s.listen.Close()
}

func (s *Server) Run() (err error) {
	if s.handler == nil {
		err = errors.New("socket handler not defined")
		return
	}
	if s.tlsConfig != nil {
		s.mu.Lock()
		s.listen, err = tls.Listen("tcp", s.address, s.tlsConfig)
		s.mu.Unlock()
		if err != nil {
			return
		}
	} else {
		addr, err := net.ResolveTCPAddr("tcp", s.address)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.listen, err = net.ListenTCP("tcp", addr)
		s.mu.Unlock()
		if err != nil {
			return err
		}

	}
	for {
		if conn, err := s.listen.Accept(); err != nil {
			return err
		} else if conn != nil {
			go s.handler(NewConnByNetConn(conn))
		}
	}
}
