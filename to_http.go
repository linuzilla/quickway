package main

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

type toHttpProxy struct {
	remoteAddress string
	verbose       bool
}

func (proxy *toHttpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	host := ``
	if u, err := url.Parse(proxy.remoteAddress); err == nil {
		host = u.Host
	}

	var client = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	proxying(w, r, client, host, proxy.remoteAddress+r.RequestURI, proxy.verbose)
}
