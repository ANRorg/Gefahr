package server

import (
	"crypto/tls"
	"testing"

	"github.com/anouar/goproxy/internal/config"
)

func TestTLSConfigRequiresTLS12AndAdvertisesHTTP2(t *testing.T) {
	cfg := new(CertificateStore).TLSConfig()
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("minimum TLS version = %x", cfg.MinVersion)
	}
	if len(cfg.NextProtos) == 0 || cfg.NextProtos[0] != "h2" {
		t.Fatalf("ALPN protocols = %v", cfg.NextProtos)
	}
}

func TestCertificateStoreRejectsMissingPair(t *testing.T) {
	store := new(CertificateStore)
	if err := store.Load(config.TLSConfig{CertFile: "missing-cert.pem", KeyFile: "missing-key.pem"}); err == nil {
		t.Fatal("expected load error")
	}
}
