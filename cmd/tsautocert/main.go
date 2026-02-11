// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/adrianosela/tsdmg"
	"github.com/adrianosela/tsdmg/pkg/logger"
	"github.com/adrianosela/tsdmg/pkg/tsautocert"
	"golang.org/x/crypto/acme/autocert"
)

type stringSlice []string

func (s *stringSlice) String() string         { return strings.Join(*s, ",") }
func (s *stringSlice) Set(value string) error { *s = append(*s, value); return nil }

const (
	statusCodeNoError = 0
	statusCodeError   = 1
)

var (
	serverURL         string
	cn                string
	sans              stringSlice
	cachedir          string
	ensureAddrRecords bool
	serveHelloWorld   bool
)

func parseFlags() {
	flag.StringVar(&serverURL, "server-url", "http://tsdmg", "TSDMG server url e.g. http://tsdmg")
	flag.StringVar(&cn, "cn", "", "Common Name (CN) for certificate")
	flag.Var(&sans, "san", "Subject Alternative Nave (SAN) for certificate (repeatable)")
	flag.StringVar(&cachedir, "cachedir", "./certcache", "File system location for certificate cache")
	flag.BoolVar(&ensureAddrRecords, "ensure-address-records", false, "Request tsdmg server to create A and AAAA to Tailscale private IPs")
	flag.BoolVar(&serveHelloWorld, "serve-hello-world", false, "Serve a hello world on port 443 using the retrieved certificate")
	flag.Parse()
}

func main() {
	os.Exit(run())
}

func run() int {
	logger := logger.New()
	ctx := context.Background()

	parseFlags()

	clientOpts := []tsdmg.Option{
		tsdmg.WithLogger(logger),

		// My laptop is already running the Tailscale desktop i.e.
		// already networked with the tsdmg server, and its node
		// name (e.g. tsdmg) resolves to its Tailscale private IP.
		tsdmg.WithSkipTailscaleNode(true),
	}

	client, err := tsdmg.NewClient(ctx, serverURL, clientOpts...)
	if err != nil {
		logger.Error("failed to initialize client", "error", err)
		return statusCodeError
	}
	defer client.Close()

	logger.Info("tsdmg client initialized")

	// Request address records (A + AAAA) if applicable.
	if ensureAddrRecords {
		createdRecords, err := client.Register(ctx)
		if err != nil {
			logger.Error("failed to initialize client", "error", err)
			return statusCodeError
		}
		logger.Info("node ensured address records (A + AAAA) via tsdmg server", "created", createdRecords)
	}

	certManagerOpts := []tsautocert.Option{
		tsautocert.WithLogger(logger),

		// Cache certificates in the filesystem to
		// avoid hitting the Let's Encrypt rate limit.
		tsautocert.WithCertificateCache(autocert.DirCache(cachedir)),
	}

	certManager, err := tsautocert.NewCertificateManager(ctx, client, "adrianos-macbook.tsdmg.net", certManagerOpts...)
	if err != nil {
		logger.Error("failed to initialize certificate manager", "error", err)
		return statusCodeError
	}
	defer certManager.Close()

	logger.Info("certificate manager initialized")

	if err := certManager.WaitForInitialCert(ctx); err != nil {
		logger.Error("failed to wait for initial certificate", "error", err)
		return statusCodeError
	}
	logger.Info("certificate manager's initial certificate is ready")

	ln, err := net.Listen("tcp", ":443")
	if err != nil {
		logger.Error("failed to start tcp listener on :443", "error", err)
		return statusCodeError
	}

	// Configure TLS listener to get certificate using tsdmg client.
	ln = tls.NewListener(ln, &tls.Config{GetCertificate: certManager.GetCertificate})
	defer ln.Close()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Run server in a goroutine if applicable
	errCh := make(chan error, 1)
	defer close(errCh)
	if serveHelloWorld {
		go func() {
			err = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Hello World!"))
			}))
			errCh <- err
		}()
	}

	// Wait for shutdown signal or error
	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down gracefully", "signal", sig.String())
		return statusCodeNoError
	case err := <-errCh:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error("server error", "error", err)
			return statusCodeError
		}
		return statusCodeError
	}
}
