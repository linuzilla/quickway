package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"log"
	"net/http"
	"time"
)

type toQuicProxy struct {
	remoteAddress string
	verbose       bool
}

func (proxy *toQuicProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
	}

	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: true,
		},
		QuicConfig: &quic.Config{},
	}
	defer roundTripper.Close()

	var client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: roundTripper,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	proxying(w, r, client, ``, proxy.remoteAddress+r.RequestURI, proxy.verbose)
}
