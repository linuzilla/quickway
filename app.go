package main

import (
	"fmt"
	"github.com/kesselborn/go-getopt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	_ "net/http/pprof"
)

func runtimeDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current frame")
	}

	return path.Dir(filename)
}

func GetCertificatePaths() (string, string) {
	certPath := runtimeDir()
	return path.Join(certPath, "cert.pem"), path.Join(certPath, "priv.key")
}

func listenOnQuicAndRedirectToHttp(remoteAddress string, bindPort int, verbose bool) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		server := http3.Server{
			Handler: &toHttpProxy{
				remoteAddress: remoteAddress,
				verbose:       verbose,
			},
			Addr:       fmt.Sprintf(":%d", bindPort),
			QuicConfig: &quic.Config{},
		}

		err := server.ListenAndServeTLS(GetCertificatePaths())

		if err != nil {
			fmt.Println(err)
		}
		wg.Done()
	}()
	wg.Wait()
}

func listenOnHttpAndRedirectViaQuic(remoteAddress string, localAddress string, port int, verbose bool) {
	s := &http.Server{
		Addr: fmt.Sprintf("%s:%d", localAddress, port),
		Handler: &toQuicProxy{
			remoteAddress: remoteAddress,
			verbose:       verbose,
		},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err := s.ListenAndServe()
	if err != nil {
		fmt.Printf("Server failed: ", err.Error())
	}
}

func main() {
	optionDefinition := getopt.Options{
		Description: `Q gateway`,
		Definitions: getopt.Definitions{
			{`backend|b`, "Backend, from Quic to HTTP", getopt.Optional | getopt.Flag, false},
			{`verbose|v`, "Verbose", getopt.Optional | getopt.Flag, false},
			{`udp-port`, "Quic UDP Port", getopt.Optional | getopt.ExampleIsDefault, 8080},
			{`port`, "HTTP TCP Port", getopt.Optional | getopt.ExampleIsDefault, 8080},
			{`remote`, "Proxy To Address", getopt.Optional | getopt.ExampleIsDefault, `127.0.0.1`},
			{`bind`, "Local binding Address", getopt.Optional | getopt.ExampleIsDefault, `127.0.0.1`},
		},
	}

	options, _, _, e := optionDefinition.ParseCommandLine()

	help, wantsHelp := options["help"]
	exitCode := 0

	if e != nil || wantsHelp {
		switch {
		case wantsHelp && help.String == "usage":
			fmt.Print(optionDefinition.Usage())
		case wantsHelp && help.String == "help":
			fmt.Print(optionDefinition.Help())
		default:
			fmt.Println("**** Error: ", e.Error(), "\n", optionDefinition.Help())
			exitCode = e.ErrorCode
		}
	} else {
		verbose := options[`verbose`].Bool

		if options[`backend`].Bool {
			listenOnQuicAndRedirectToHttp(options[`remote`].String, int(options[`udp-port`].Int), verbose)
		} else {
			remoteAddress := fmt.Sprintf("https://%s:%d", options[`remote`].String, options[`udp-port`].Int)
			listenOnHttpAndRedirectViaQuic(remoteAddress, options[`bind`].String, int(options[`port`].Int), verbose)
		}
	}
	os.Exit(exitCode)
}
