package server

import (
	"crypto/tls"
	"time"
)

type OptionFn func(*Server)

func WithTLSConfig(cfg *tls.Config) OptionFn {
	return func(s *Server) {
		s.tlsConfig = cfg
	}
}

func WithReadTimeout(readtimeout time.Duration) OptionFn {
	return func(s *Server) {
		s.readTimeout = readtimeout
	}
}

func WithWriteTimeout(writetimeout time.Duration) OptionFn {
	return func(s *Server) {
		s.writeTimeout = writetimeout
	}
}
