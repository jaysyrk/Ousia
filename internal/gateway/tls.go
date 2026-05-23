package gateway

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/jaysyrk/ousia/pkg/types"
)

func buildTLSConfig(tlsCfg *types.TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("tls: failed to load cert/key pair: %w", err)
	}

	return &tls.Config{
		Certificates:	[]tls.Certificate{cert},
		MinVersion:	tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}, nil
}

func redirectToHTTPS(tlsAddr string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == "" {
			host = "localhost" + tlsAddr
		}
		target := "https://" + host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

func newTLSServer(addr string, handler http.Handler, tlsCfg *tls.Config) *http.Server {
	return &http.Server{
		Addr:		addr,
		Handler:	handler,
		TLSConfig:	tlsCfg,
		ReadTimeout:	15 * time.Second,
		WriteTimeout:	60 * time.Second,
		IdleTimeout:	120 * time.Second,
	}
}
