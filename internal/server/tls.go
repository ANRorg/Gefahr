// Package server owns public/admin HTTP server lifecycle and TLS policy.
package server

import (
	"crypto/tls"
	"fmt"
	"sync/atomic"

	"github.com/anouar/goproxy/internal/config"
)

// CertificateStore atomically serves a reloadable TLS certificate.
type CertificateStore struct {
	certificate atomic.Pointer[tls.Certificate]
}

// Load validates and publishes the configured PEM certificate pair.
func (s *CertificateStore) Load(cfg config.TLSConfig) error {
	certificate, err := LoadCertificate(cfg)
	if err != nil {
		return err
	}
	s.Publish(certificate)
	return nil
}

// LoadCertificate validates a PEM pair without publishing it.
func LoadCertificate(cfg config.TLSConfig) (*tls.Certificate, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS certificate: %w", err)
	}
	return &certificate, nil
}

// Publish atomically replaces the certificate used by new handshakes.
func (s *CertificateStore) Publish(certificate *tls.Certificate) { s.certificate.Store(certificate) }

// TLSConfig returns a server policy that rejects protocols older than TLS 1.2
// and reads the active certificate at handshake time.
func (s *CertificateStore) TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			certificate := s.certificate.Load()
			if certificate == nil {
				return nil, fmt.Errorf("TLS certificate is not loaded")
			}
			return certificate, nil
		},
	}
}
