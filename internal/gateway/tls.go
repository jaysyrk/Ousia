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
		MinVersion:	tls.VersionTLS13,
	}, nil
}

func redirectToHTTPS(tlsAddr string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == "" {
			host = "localhost" + tlsAddr
		}
		target := "https://" + host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
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
