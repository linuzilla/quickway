package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var cookiePattern = regexp.MustCompile("(?i)set-cookie")

var hopHeaderMap = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true, // canonicalized version of "TE"
	"Trailers":            true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
	"Host":                true,
	//"Origin":              true,
}

func copyHeader(dst, src http.Header, forwardRequest bool, verbose bool) {
	for k, vv := range src {
		if _, ok := hopHeaderMap[k]; ok && forwardRequest {
			continue
		}
		for _, v := range vv {
			if verbose && cookiePattern.MatchString(k) {
				if forwardRequest {
					fmt.Printf("-> %s: %s\n", k, v)
				} else {
					fmt.Printf("<- %s: %s\n", k, v)
				}
			}
			if k == `Host` {
				fmt.Printf("Host: %s\n", v)
			}
			if k == "Location" {
				if u, err := url.Parse(v); err == nil {
					dst.Add(k, u.RequestURI())
				} else {
					dst.Add(k, v)
				}
				if verbose {
					if forwardRequest {
						fmt.Printf("-> %s: %s\n", k, v)
					} else {
						fmt.Printf("<- %s: %s\n", k, v)
					}
				}
			} else {
				dst.Add(k, v)
			}
		}
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

func proxying(w http.ResponseWriter, r *http.Request, client *http.Client, host string, urlString string, verbose bool) {
	requestBody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	//fmt.Println(r.Method, urlString)
	//if len(requestBody) > 0 {
	//	fmt.Println("DATA")
	//	fmt.Println(string(requestBody))
	//}

	bodyReader := bytes.NewReader(requestBody)

	if req, err := http.NewRequest(r.Method, urlString, bodyReader); err != nil {
		fmt.Println(err)
	} else {
		copyHeader(req.Header, r.Header, true, verbose)
		if host != `` {
			req.Header.Add(`Host`, host)
		}

		if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			appendHostToXForwardHeader(r.Header, clientIP)
		}

		if response, err := client.Do(req); err != nil {
			fmt.Println(err)
		} else {
			defer response.Body.Close()

			if verbose {
				fmt.Printf("%s %s [%d]\n", r.Method, urlString, response.StatusCode)
				if verbose && len(requestBody) > 0 {
					fmt.Println("DATA")
					fmt.Println(string(requestBody))
				}
			}

			copyHeader(w.Header(), response.Header, false, verbose)
			w.WriteHeader(response.StatusCode)

			io.Copy(w, response.Body)
		}
	}
}
